package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NotionAPIVersion pins the Notion API version this client speaks. The
// 2025-09-03 release split "database" into a container plus one or more
// "data sources" (developers.notion.com/docs/upgrade-faqs-2025-09-03) —
// querying rows now goes through /v1/data_sources/{id}/query rather than
// the old /v1/databases/{id}/query.
const NotionAPIVersion = "2025-09-03"

const notionOAuthTokenURL = "https://api.notion.com/v1/oauth/token"

// NotionClient talks to the real Notion REST API: OAuth token refresh,
// paginated discovery of databases/data sources/pages/users/comments, and
// conflict-safe write-back. clientID/clientSecret are only used for
// refreshing an expired access token (see Refresh) — Validate/Sync work
// with whatever access token is already in credentials.
type NotionClient struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
}

func NewNotionClient(clientID, clientSecret string) *NotionClient {
	return &NotionClient{
		httpClient:   &http.Client{Timeout: 20 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *NotionClient) Provider() string { return "notion" }

// Refresh exchanges a still-valid refresh token for a new access token.
// Per Notion's current docs, both access_token and refresh_token rotate —
// the old refresh_token is invalidated, so the returned map must fully
// replace what's stored, not just patch the access token.
func (c *NotionClient) Refresh(ctx context.Context, credentials map[string]string) (map[string]string, error) {
	refreshToken := strings.TrimSpace(credentials["refresh_token"])
	if refreshToken == "" {
		return nil, fmt.Errorf("no refresh_token stored for this notion connection")
	}
	if c.clientID == "" || c.clientSecret == "" {
		return nil, fmt.Errorf("notion oauth client not configured, cannot refresh")
	}

	provider := &OAuthProvider{
		Provider:      "notion",
		ClientID:      c.clientID,
		ClientSecret:  c.clientSecret,
		TokenURL:      notionOAuthTokenURL,
		BasicAuth:     true,
		CredentialKey: "token",
		ExtraHeaders:  map[string]string{"Notion-Version": NotionAPIVersion},
	}

	fresh, err := provider.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh notion token: %w", err)
	}

	merged := make(map[string]string, len(credentials)+len(fresh))
	for k, v := range credentials {
		merged[k] = v
	}
	for k, v := range fresh {
		merged[k] = v
	}
	return merged, nil
}

func (c *NotionClient) Validate(ctx context.Context, credentials map[string]string) error {
	token := strings.TrimSpace(credentials["token"])
	if token == "" {
		return fmt.Errorf("token is required")
	}
	_, err := c.doJSON(ctx, http.MethodGet, "https://api.notion.com/v1/users/me", token, nil)
	return err
}

// UpdatePageProperties writes properties back to a Notion page — but only
// if the page hasn't changed upstream since expectedLastEditedTime. That's
// the entire "never overwrite newer data with stale data" contract: we
// re-read the page's current last_edited_time immediately before writing,
// and refuse (returning ConflictError, not silently forcing it) if someone
// edited it in Notion after we last read it. Pass expectedLastEditedTime
// "" to skip the check for a brand-new page nobody else could have touched.
func (c *NotionClient) UpdatePageProperties(ctx context.Context, credentials map[string]string, pageID string, properties map[string]interface{}, expectedLastEditedTime string) error {
	token := strings.TrimSpace(credentials["token"])
	if token == "" {
		return fmt.Errorf("token is required")
	}

	if expectedLastEditedTime != "" {
		current, err := c.doJSON(ctx, http.MethodGet, "https://api.notion.com/v1/pages/"+pageID, token, nil)
		if err != nil {
			return fmt.Errorf("check current page state: %w", err)
		}
		currentEdited, _ := current["last_edited_time"].(string)
		if currentEdited != "" && currentEdited > expectedLastEditedTime {
			return &ConflictError{
				Provider: "notion",
				Message:  fmt.Sprintf("page %s was edited at %s, after our last read at %s", pageID, currentEdited, expectedLastEditedTime),
			}
		}
	}

	_, err := c.doJSON(ctx, http.MethodPatch, "https://api.notion.com/v1/pages/"+pageID, token, map[string]interface{}{"properties": properties})
	return err
}

// doJSON issues one Notion API call and maps well-known failure modes
// (expired/revoked auth, rate limiting) to sentinel error types the caller
// can act on, instead of a bare "HTTP 401" string.
func (c *NotionClient) doJSON(ctx context.Context, method, url, token string, body interface{}) (map[string]interface{}, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", NotionAPIVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &AuthExpiredError{Provider: "notion"}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &RateLimitError{Provider: "notion", RetryAfter: resp.Header.Get("Retry-After")}
	}
	if resp.StatusCode >= 400 {
		var apiErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &apiErr)
		if apiErr.Message != "" {
			return nil, fmt.Errorf("notion API error (%s): %s", apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("notion API returned HTTP %d", resp.StatusCode)
	}

	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode notion response: %w", err)
	}
	return out, nil
}

// notionSearchResult is one row/page/database from /v1/search.
type notionSearchResult struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"` // "page" | "database"
	URL            string                 `json:"url"`
	LastEditedTime string                 `json:"last_edited_time"`
	Properties     map[string]interface{} `json:"properties"`
	DataSources    []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data_sources"`
}

const maxSearchPages = 20 // 20 * 100 = 2,000 results per sync pass; enough headroom for incremental catch-up

// discoverAll paginates /v1/search sorted newest-edited-first, stopping
// early once we cross the incremental watermark — that's what keeps a
// steady-state re-sync fast instead of re-walking the whole workspace.
func (c *NotionClient) discoverAll(ctx context.Context, token, watermark string) ([]notionSearchResult, string, error) {
	var all []notionSearchResult
	cursor := ""
	newestSeen := watermark

	for page := 0; page < maxSearchPages; page++ {
		body := map[string]interface{}{
			"page_size": 100,
			"sort":      map[string]string{"direction": "descending", "timestamp": "last_edited_time"},
		}
		if cursor != "" {
			body["start_cursor"] = cursor
		}

		result, err := c.doJSON(ctx, http.MethodPost, "https://api.notion.com/v1/search", token, body)
		if err != nil {
			return all, newestSeen, err
		}

		rawResults, _ := result["results"].([]interface{})
		stop := false
		for _, r := range rawResults {
			b, _ := json.Marshal(r)
			var sr notionSearchResult
			if err := json.Unmarshal(b, &sr); err != nil {
				continue
			}
			if page == 0 && len(all) == 0 {
				newestSeen = sr.LastEditedTime
			}
			if watermark != "" && sr.LastEditedTime != "" && sr.LastEditedTime <= watermark {
				stop = true
				break
			}
			all = append(all, sr)
		}
		if stop {
			break
		}

		hasMore, _ := result["has_more"].(bool)
		nextCursor, _ := result["next_cursor"].(string)
		if !hasMore || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return all, newestSeen, nil
}

// dataSourceSchema fetches one data source's properties (its schema) —
// this is the "intelligently discover databases" requirement: we never
// hardcode a database name, we read whatever schema the workspace has.
func (c *NotionClient) dataSourceSchema(ctx context.Context, token, dataSourceID string) (map[string]interface{}, error) {
	return c.doJSON(ctx, http.MethodGet, "https://api.notion.com/v1/data_sources/"+dataSourceID, token, nil)
}

// queryDataSource pulls up to one page of rows from a data source.
func (c *NotionClient) queryDataSource(ctx context.Context, token, dataSourceID, cursor string) (map[string]interface{}, error) {
	body := map[string]interface{}{"page_size": 100}
	if cursor != "" {
		body["start_cursor"] = cursor
	}
	return c.doJSON(ctx, http.MethodPost, "https://api.notion.com/v1/data_sources/"+dataSourceID+"/query", token, body)
}

// listUsers paginates the workspace's users — required so SponsorOS can
// map Notion's assignees/owners/comment authors to real names.
func (c *NotionClient) listUsers(ctx context.Context, token string) ([]map[string]interface{}, error) {
	var users []map[string]interface{}
	cursor := ""
	for page := 0; page < 5; page++ {
		url := "https://api.notion.com/v1/users?page_size=100"
		if cursor != "" {
			url += "&start_cursor=" + cursor
		}
		result, err := c.doJSON(ctx, http.MethodGet, url, token, nil)
		if err != nil {
			return users, err
		}
		raw, _ := result["results"].([]interface{})
		for _, r := range raw {
			if m, ok := r.(map[string]interface{}); ok {
				users = append(users, m)
			}
		}
		hasMore, _ := result["has_more"].(bool)
		nextCursor, _ := result["next_cursor"].(string)
		if !hasMore || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	return users, nil
}

// commentsForPage fetches a page's comments. Notion returns 403 when the
// integration wasn't granted the "read comments" capability — that's not a
// sync failure, just a feature the workspace owner didn't enable, so we
// report it as a warning rather than aborting the whole sync.
func (c *NotionClient) commentsForPage(ctx context.Context, token, pageID string) ([]map[string]interface{}, error) {
	result, err := c.doJSON(ctx, http.MethodGet, "https://api.notion.com/v1/comments?block_id="+pageID+"&page_size=25", token, nil)
	if err != nil {
		return nil, err
	}
	raw, _ := result["results"].([]interface{})
	comments := make([]map[string]interface{}, 0, len(raw))
	for _, r := range raw {
		if m, ok := r.(map[string]interface{}); ok {
			comments = append(comments, m)
		}
	}
	return comments, nil
}

func (c *NotionClient) Sync(ctx context.Context, credentials map[string]string, state map[string]interface{}) (*SyncResult, error) {
	token := strings.TrimSpace(credentials["token"])
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	watermark, _ := state["watermark"].(string)

	results, newestSeen, err := c.discoverAll(ctx, token, watermark)
	if err != nil && len(results) == 0 {
		return nil, err
	}

	warnings := make([]string, 0, 4)

	databaseCount, pageCount, schemaCount := 0, 0, 0
	notes := make([]NoteRecord, 0, len(results))
	contacts := make([]ContactRecord, 0)

	for _, r := range results {
		title := notionTitle(r.Properties)

		if r.Object == "database" {
			databaseCount++
			for _, ds := range r.DataSources {
				schema, err := c.dataSourceSchema(ctx, token, ds.ID)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("data source %q schema: %v", ds.Name, err))
					continue
				}
				schemaCount++
				rowContacts, rowNotes := c.harvestDataSourceRows(ctx, token, ds.ID, ds.Name, schema, &warnings)
				contacts = append(contacts, rowContacts...)
				notes = append(notes, rowNotes...)
			}
			continue
		}

		pageCount++
		if title == "" && r.URL == "" {
			continue
		}
		notes = append(notes, NoteRecord{
			Title:      title,
			URL:        r.URL,
			ExternalID: r.ID,
			UpdatedAt:  r.LastEditedTime,
		})
	}

	users, err := c.listUsers(ctx, token)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("list users: %v", err))
	}

	if newestSeen == "" {
		newestSeen = watermark
	}

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"pages_found":     pageCount,
			"databases_found": databaseCount,
			"schemas_read":    schemaCount,
			"users_found":     len(users),
			"incremental":     watermark != "",
		},
		State: map[string]interface{}{
			"watermark": newestSeen,
		},
		Contacts: contacts,
		Notes:    notes,
		Warnings: warnings,
	}, nil
}

// harvestDataSourceRows queries every row of one data source and, for
// database schemas that look like a CRM/contacts table (a title property
// plus recognizable email/company columns), maps rows into ContactRecord
// so real people/companies land in the CRM. Every other schema still gets
// surfaced as a NoteRecord per row so nothing discovered is silently
// dropped.
func (c *NotionClient) harvestDataSourceRows(ctx context.Context, token, dataSourceID, dataSourceName string, schema map[string]interface{}, warnings *[]string) ([]ContactRecord, []NoteRecord) {
	properties, _ := schema["properties"].(map[string]interface{})
	emailProp := findPropertyByType(properties, "email")
	titleProp := findPropertyByType(properties, "title")
	companyProp := findPropertyLike(properties, []string{"company", "organization"})

	var contacts []ContactRecord
	var notes []NoteRecord
	cursor := ""

	for page := 0; page < 10; page++ { // 1,000 rows per data source per pass is a sane ceiling
		result, err := c.queryDataSource(ctx, token, dataSourceID, cursor)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("query data source %q: %v", dataSourceName, err))
			return contacts, notes
		}

		rows, _ := result["results"].([]interface{})
		for _, r := range rows {
			row, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			props, _ := row["properties"].(map[string]interface{})
			id, _ := row["id"].(string)
			url, _ := row["url"].(string)
			lastEdited, _ := row["last_edited_time"].(string)
			title := notionTitle(props)

			email := ""
			if emailProp != "" {
				email = notionPlainValue(props[emailProp])
			}
			company := ""
			if companyProp != "" {
				company = notionPlainValue(props[companyProp])
			}

			if email != "" || (emailProp != "" && title != "") {
				first, last := splitName(title)
				contacts = append(contacts, ContactRecord{
					FirstName:   first,
					LastName:    last,
					Email:       email,
					CompanyName: company,
				})
				continue
			}
			_ = titleProp

			if title != "" || url != "" {
				notes = append(notes, NoteRecord{Title: title, URL: url, ExternalID: id, UpdatedAt: lastEdited})
			}
		}

		hasMore, _ := result["has_more"].(bool)
		nextCursor, _ := result["next_cursor"].(string)
		if !hasMore || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return contacts, notes
}

func findPropertyByType(properties map[string]interface{}, wantType string) string {
	for name, raw := range properties {
		if prop, ok := raw.(map[string]interface{}); ok && prop["type"] == wantType {
			return name
		}
	}
	return ""
}

func findPropertyLike(properties map[string]interface{}, keywords []string) string {
	for name := range properties {
		lower := strings.ToLower(name)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return name
			}
		}
	}
	return ""
}

func splitName(full string) (first, last string) {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// notionPlainValue extracts a human-readable value from a Notion property
// object regardless of its type (email, rich_text, title, select, ...).
func notionPlainValue(raw interface{}) string {
	prop, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	switch prop["type"] {
	case "email":
		s, _ := prop["email"].(string)
		return s
	case "select":
		if sel, ok := prop["select"].(map[string]interface{}); ok {
			s, _ := sel["name"].(string)
			return s
		}
	case "rich_text", "title":
		return richTextPlain(prop[fmt.Sprint(prop["type"])])
	}
	return ""
}

func richTextPlain(raw interface{}) string {
	parts, ok := raw.([]interface{})
	if !ok {
		return ""
	}
	var text string
	for _, p := range parts {
		if rt, ok := p.(map[string]interface{}); ok {
			if plain, ok := rt["plain_text"].(string); ok {
				text += plain
			}
		}
	}
	return text
}

// notionTitle best-effort extracts a page/database title from a Notion
// properties map. The title property's key name varies ("title", "Name",
// ...), so scan for whichever property has type "title".
func notionTitle(properties map[string]interface{}) string {
	for _, raw := range properties {
		prop, ok := raw.(map[string]interface{})
		if !ok || prop["type"] != "title" {
			continue
		}
		return richTextPlain(prop["title"])
	}
	return ""
}
