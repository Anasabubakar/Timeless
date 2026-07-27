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

const notionAPIVersion = "2022-06-28"

// NotionClient is a native fallback integration used when a workspace
// can't be reached via Zapier. It talks directly to the Notion REST API.
type NotionClient struct {
	httpClient *http.Client
}

func NewNotionClient() *NotionClient {
	return &NotionClient{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *NotionClient) Provider() string { return "notion" }

func (c *NotionClient) Validate(ctx context.Context, credentials map[string]string) error {
	token := strings.TrimSpace(credentials["token"])
	if token == "" {
		return fmt.Errorf("token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.notion.com/v1/users/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notion request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notion rejected the token (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (c *NotionClient) Sync(ctx context.Context, credentials map[string]string) (*SyncResult, error) {
	token := strings.TrimSpace(credentials["token"])

	body, err := json.Marshal(map[string]interface{}{"page_size": 25})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.notion.com/v1/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion search returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Results []map[string]interface{} `json:"results"`
		HasMore bool                     `json:"has_more"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode notion search response: %w", err)
	}

	notes := make([]NoteRecord, 0, len(result.Results))
	for _, r := range result.Results {
		url, _ := r["url"].(string)
		title := notionTitle(r)
		if title == "" && url == "" {
			continue
		}
		notes = append(notes, NoteRecord{Title: title, URL: url})
	}

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"pages_found": len(result.Results),
			"has_more":    result.HasMore,
		},
		Notes: notes,
	}, nil
}

// notionTitle best-effort extracts a page/database title from a Notion
// search result object. The title property's key name varies ("title",
// "Name", ...), so scan for whichever property has type "title".
func notionTitle(result map[string]interface{}) string {
	properties, _ := result["properties"].(map[string]interface{})
	for _, raw := range properties {
		prop, ok := raw.(map[string]interface{})
		if !ok || prop["type"] != "title" {
			continue
		}
		titleParts, _ := prop["title"].([]interface{})
		var text string
		for _, part := range titleParts {
			rt, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			if plain, ok := rt["plain_text"].(string); ok {
				text += plain
			}
		}
		if text != "" {
			return text
		}
	}
	return ""
}
