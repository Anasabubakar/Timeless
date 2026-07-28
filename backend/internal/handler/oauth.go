package handler

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/integration"
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
	providers map[string]*integration.OAuthProvider
}

func NewOAuthHandler(cfg *config.Config, rdb *redis.Client, svc *service.IntegrationService) *OAuthHandler {
	return &OAuthHandler{
		cfg: cfg,
		rdb: rdb,
		svc: svc,
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
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return h.frontendRedirect(c, url.Values{"error": {"invalid session, please sign in again"}})
	}
	orgID, _ := claims["org_id"].(string)
	userID, _ := claims["sub"].(string)
	if orgID == "" || userID == "" {
		return h.frontendRedirect(c, url.Values{"error": {"invalid session, please sign in again"}})
	}

	state := uuid.NewString()
	payload, _ := json.Marshal(oauthState{OrgID: orgID, UserID: userID, Provider: providerName})
	if err := h.rdb.Set(c.Context(), "oauth_state:"+state, payload, 10*time.Minute).Err(); err != nil {
		return h.frontendRedirect(c, url.Values{"error": {"could not start oauth flow"}})
	}

	return c.Redirect().Status(fiber.StatusFound).To(provider.AuthorizeRedirect(h.redirectURI(), state))
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

	credentials, err := provider.Exchange(c.Context(), c.Query("code"), h.redirectURI())
	if err != nil {
		return h.frontendRedirect(c, url.Values{"error": {err.Error()}, "provider": {state.Provider}})
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
		return h.frontendRedirect(c, url.Values{"error": {err.Error()}, "provider": {state.Provider}})
	}

	return h.frontendRedirect(c, url.Values{"connected": {state.Provider}})
}
