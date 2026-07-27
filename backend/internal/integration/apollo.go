package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ApolloClient is a native fallback integration for prospecting data
// (companies, contacts) that Zapier can't surface. It talks directly to
// the Apollo.io REST API.
type ApolloClient struct {
	httpClient *http.Client
}

func NewApolloClient() *ApolloClient {
	return &ApolloClient{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *ApolloClient) Provider() string { return "apollo" }

// apolloAuth applies either an OAuth bearer token or a static API key,
// so both the OAuth connect flow and manual key entry work.
func apolloAuth(req *http.Request, credentials map[string]string) error {
	if token := strings.TrimSpace(credentials["access_token"]); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	if apiKey := strings.TrimSpace(credentials["api_key"]); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
		return nil
	}
	return fmt.Errorf("api_key or access_token is required")
}

func (c *ApolloClient) Validate(ctx context.Context, credentials map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.apollo.io/api/v1/auth/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	if err := apolloAuth(req, credentials); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apollo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apollo rejected the api key (HTTP %d)", resp.StatusCode)
	}

	var health struct {
		Healthy    bool `json:"healthy"`
		IsLoggedIn bool `json:"is_logged_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("decode apollo health response: %w", err)
	}
	if !health.Healthy || !health.IsLoggedIn {
		return fmt.Errorf("apollo api key is not active")
	}
	return nil
}

func (c *ApolloClient) Sync(ctx context.Context, credentials map[string]string) (*SyncResult, error) {
	body, err := json.Marshal(map[string]interface{}{"page": 1, "per_page": 25})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.apollo.io/api/v1/mixed_people/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	if err := apolloAuth(req, credentials); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apollo search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apollo search returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		People []struct {
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Email        string `json:"email"`
			Title        string `json:"title"`
			LinkedinURL  string `json:"linkedin_url"`
			Organization struct {
				Name          string `json:"name"`
				WebsiteURL    string `json:"website_url"`
				PrimaryDomain string `json:"primary_domain"`
			} `json:"organization"`
		} `json:"people"`
		Pagination struct {
			TotalEntries int `json:"total_entries"`
		} `json:"pagination"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode apollo search response: %w", err)
	}

	contacts := make([]ContactRecord, 0, len(result.People))
	for _, p := range result.People {
		if p.Organization.Name == "" && p.FirstName == "" && p.LastName == "" {
			continue
		}
		contacts = append(contacts, ContactRecord{
			FirstName:      p.FirstName,
			LastName:       p.LastName,
			Email:          p.Email,
			Title:          p.Title,
			LinkedinURL:    p.LinkedinURL,
			CompanyName:    p.Organization.Name,
			CompanyDomain:  p.Organization.PrimaryDomain,
			CompanyWebsite: p.Organization.WebsiteURL,
		})
	}

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"contacts_found": len(result.People),
			"total_entries":  result.Pagination.TotalEntries,
		},
		Contacts: contacts,
	}, nil
}
