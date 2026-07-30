package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/security"
	"github.com/timeless/backend/internal/service"
)

// OAuthHandler runs the browser-redirect OAuth flow for integrations.
// Both endpoints are public routes: the start leg authenticates via a JWT
// passed as a query param (browsers can't attach Authorization headers to
// top-level navigations), and the callback leg authenticates via the state
// value minted at start time.
type OAuthHandler struct {
	cfg       *config.Config
	rdb       *redis.Client
	svc       *service.IntegrationService
	db        *gorm.DB
	keyring   *security.JWTKeyring
	providers map[string]*integration.OAuthProvider
}

func NewOAuthHandler(cfg *config.Config, rdb *redis.Client, svc *service.IntegrationService, db *gorm.DB) *OAuthHandler {
	return &OAuthHandler{
		cfg:     cfg,
		rdb:     rdb,
		svc:     svc,
		db:      db,
		keyring: security.NewJWTKeyring(cfg.JWTSecret, cfg.JWTSecretPrevious...),
		providers: map[string]*integration.OAuthProvider{
			"notion": {
				Provider:      "notion",
				ClientID:      cfg.NotionClientID,
				ClientSecret:  cfg.NotionClientSecret,
				AuthorizeURL:  "https://api.notion.com/v1/oauth/authorize",
				TokenURL:      "https://api.notion.com/v1/oauth/token",
				BasicAuth:     true,
				CredentialKey: "token",
				ExtraHeaders:  map[string]string{"Notion-Version": integration.NotionAPIVersion},
			},
			"apollo": {
				Provider:      "apollo",
				ClientID:      cfg.ApolloClientID,
				ClientSecret:  cfg.ApolloClientSecret,
				AuthorizeURL:  "https://app.apollo.io/#/oauth/authorize",
				TokenURL:      "https://app.apollo.io/api/v1/oauth/token",
				CredentialKey: "access_token",
			},
			// Zapier MCP has no third-party OAuth app registration — it only
			// supports a user generating their own personal MCP Server URL
			// at mcp.zapier.com and pasting it in (see integration/zapier.go).
			// Intentionally not registered here.
		},
	}
}

type oauthState struct {
	OrgID    string `json:"org_id"`
	UserID   string `json:"user_id"`
	Provider string `json:"provider"`
	// CodeVerifier is only set (and only meaningful) for providers with
	// PKCE enabled — see OAuthProvider.PKCE.
	CodeVerifier string `json:"code_verifier,omitempty"`
}

func (h *OAuthHandler) redirectURI() string {
	return h.cfg.APIPublicURL + "/api/v1/integrations/oauth/callback"
}

func (h *OAuthHandler) frontendRedirect(c fiber.Ctx, params url.Values) error {
	return c.Redirect().Status(fiber.StatusFound).To(h.cfg.FrontendURL + "/integrations?" + params.Encode())
}

// Start: GET /integrations/:provider/oauth/start?token=<jwt>
func (h *OAuthHandler) Start(c fiber.Ctx) error {
	providerName := c.Params("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		return h.frontendRedirect(c, url.Values{"error": {"unsupported provider"}})
	}
	if !provider.Configured() {
		return h.frontendRedirect(c, url.Values{
			"error":    {"oauth_not_configured"},
			"provider": {providerName},
		})
	}

	tokenStr := c.Query("token")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// Matches middleware.AuthMiddleware's keyfunc: reject anything
		// that isn't HMAC-signed (defense against algorithm-confusion
		// attacks — a keyfunc that ignores t.Method and just returns a
		// key will happily "verify" a token forged with a different
		// algorithm the key wasn't meant for), and resolve the signing
		// key via the same kid-aware keyring the main auth middleware
		// and AuthService use. Without this, a token signed under a
		// just-rotated-out JWT_SECRET_PREVIOUS key (which the main API
		// still accepts) would be rejected here — this endpoint would
		// silently break for anyone who happened to be mid-session
		// during a key rotation.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		key, ok := h.keyring.Key(kid)
		if !ok {
			return nil, errors.New("unknown signing key")
		}
		return key, nil
	})
	if err != nil || !token.Valid {
		return h.frontendRedirect(c, url.Values{"error": {"invalid session, please sign in again"}})
	}
	if claims["type"] != "access" {
		return h.frontendRedirect(c, url.Values{"error": {"invalid session, please sign in again"}})
	}
	orgID, _ := claims["org_id"].(string)
	userID, _ := claims["sub"].(string)
	if orgID == "" || userID == "" {
		return h.frontendRedirect(c, url.Values{"error": {"invalid session, please sign in again"}})
	}

	state := uuid.NewString()
	var codeVerifier string
	if provider.PKCE {
		codeVerifier, err = integration.GeneratePKCEVerifier()
		if err != nil {
			return h.frontendRedirect(c, url.Values{"error": {"could not start oauth flow"}})
		}
	}

	payload, _ := json.Marshal(oauthState{OrgID: orgID, UserID: userID, Provider: providerName, CodeVerifier: codeVerifier})
	if err := h.rdb.Set(c.Context(), "oauth_state:"+state, payload, 10*time.Minute).Err(); err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"could not start oauth flow"}})
	}

	return c.Redirect().Status(fiber.StatusFound).To(provider.AuthorizeRedirect(h.redirectURI(), state, codeVerifier))
}

// Callback: GET /integrations/oauth/callback?code=...&state=...
func (h *OAuthHandler) Callback(c fiber.Ctx) error {
	if errMsg := c.Query("error"); errMsg != "" {
		return h.frontendRedirect(c, url.Values{"error": {errMsg}})
	}

	stateKey := "oauth_state:" + c.Query("state")
	raw, err := h.rdb.GetDel(c.Context(), stateKey).Result()
	if err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"oauth session expired, please try again"}})
	}
	var state oauthState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"invalid oauth state"}})
	}

	provider, ok := h.providers[state.Provider]
	if !ok {
		return h.frontendRedirect(c, url.Values{"error": {"unsupported provider"}})
	}

	credentials, err := provider.Exchange(c.Context(), c.Query("code"), h.redirectURI(), state.CodeVerifier)
	if err != nil {
		// The provider's token-exchange error can carry internal
		// details (request/response fragments) that don't belong in a
		// URL a browser will keep in history and send as a Referer —
		// log it server-side, send the user a generic message.
		log.Printf("oauth: token exchange failed for provider %s: %v", state.Provider, err)
		return h.frontendRedirect(c, url.Values{"error": {"could not connect to " + state.Provider}, "provider": {state.Provider}})
	}

	orgID, err := uuid.Parse(state.OrgID)
	if err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"invalid oauth state"}})
	}
	userID, err := uuid.Parse(state.UserID)
	if err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"invalid oauth state"}})
	}

	if _, err := h.svc.Connect(c.Context(), orgID, userID, state.Provider, service.ConnectInput{Credentials: credentials}); err != nil {
		log.Printf("oauth: failed to persist connection for org %s provider %s: %v", orgID, state.Provider, err)
		return h.frontendRedirect(c, url.Values{"error": {"could not connect to " + state.Provider}, "provider": {state.Provider}})
	}

	middleware.LogSecurityEvent(h.db, orgID, &userID, "integration", "integration_connected",
		state.Provider+" connected via OAuth", c.IP(), map[string]string{"provider": state.Provider})

	return h.frontendRedirect(c, url.Values{"connected": {state.Provider}})
}
