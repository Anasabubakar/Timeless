package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthProvider describes one provider's OAuth 2.0 authorization-code flow.
// Client credentials come from env config; a provider with an empty ClientID
// is treated as "OAuth not configured" and callers should fall back to the
// manual credential connect flow.
type OAuthProvider struct {
	Provider     string
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	Scope        string
	// BasicAuth: send client credentials via HTTP Basic (Notion style)
	// instead of in the form body.
	BasicAuth bool
	// CredentialKey is the key the exchanged access token is stored under in
	// Integration.Credentials, matching what each Client's Validate/Sync expects.
	CredentialKey string
	// ExtraHeaders are sent on every token request (e.g. Notion requires
	// Notion-Version on API calls; harmless to include on the token
	// endpoint too since Notion's own examples do).
	ExtraHeaders map[string]string
}

func (p *OAuthProvider) Configured() bool {
	return p.ClientID != "" && p.ClientSecret != ""
}

// AuthorizeRedirect builds the provider authorization URL for this flow.
func (p *OAuthProvider) AuthorizeRedirect(redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	if p.Scope != "" {
		q.Set("scope", p.Scope)
	}
	if p.Provider == "notion" {
		q.Set("owner", "user")
	}
	sep := "?"
	if strings.Contains(p.AuthorizeURL, "?") {
		sep = "&"
	}
	return p.AuthorizeURL + sep + q.Encode()
}

// Exchange swaps an authorization code for an access token and returns the
// credentials map to store on the Integration record.
func (p *OAuthProvider) Exchange(ctx context.Context, code, redirectURI string) (map[string]string, error) {
	return p.token(ctx, map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": redirectURI,
	})
}

// Refresh swaps a still-valid refresh token for a new access token. Per
// Notion's current docs (developers.notion.com/reference/refresh-a-token),
// the response rotates BOTH tokens — the old refresh token stops working,
// so callers must persist the newly returned refresh_token too, not just
// the access_token.
func (p *OAuthProvider) Refresh(ctx context.Context, refreshToken string) (map[string]string, error) {
	return p.token(ctx, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
}

func (p *OAuthProvider) token(ctx context.Context, grantParams map[string]string) (map[string]string, error) {
	var body io.Reader
	var contentType string

	if p.BasicAuth {
		payload, err := json.Marshal(grantParams)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(payload))
		contentType = "application/json"
	} else {
		form := url.Values{}
		for k, v := range grantParams {
			form.Set(k, v)
		}
		form.Set("client_id", p.ClientID)
		form.Set("client_secret", p.ClientSecret)
		body = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if p.BasicAuth {
		basic := base64.StdEncoding.EncodeToString([]byte(p.ClientID + ":" + p.ClientSecret))
		req.Header.Set("Authorization", "Basic "+basic)
	}
	for k, v := range p.ExtraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s token request: %w", p.Provider, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s token request returned HTTP %d: %s", p.Provider, resp.StatusCode, string(respBody))
	}

	var token struct {
		AccessToken   string `json:"access_token"`
		RefreshToken  string `json:"refresh_token"`
		WorkspaceID   string `json:"workspace_id"`
		WorkspaceName string `json:"workspace_name"`
		WorkspaceIcon string `json:"workspace_icon"`
		BotID         string `json:"bot_id"`
	}
	if err := json.Unmarshal(respBody, &token); err != nil {
		return nil, fmt.Errorf("%s token decode: %w", p.Provider, err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("%s token request returned no access_token", p.Provider)
	}

	credentials := map[string]string{p.CredentialKey: token.AccessToken}
	if token.RefreshToken != "" {
		credentials["refresh_token"] = token.RefreshToken
	}
	if token.WorkspaceID != "" {
		credentials["workspace_id"] = token.WorkspaceID
	}
	if token.WorkspaceName != "" {
		credentials["workspace_name"] = token.WorkspaceName
	}
	if token.WorkspaceIcon != "" {
		credentials["workspace_icon"] = token.WorkspaceIcon
	}
	if token.BotID != "" {
		credentials["bot_id"] = token.BotID
	}
	return credentials, nil
}
