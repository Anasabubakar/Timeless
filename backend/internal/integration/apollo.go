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

// ApolloClient talks to the real Apollo.io REST API for prospecting data:
// people/company search, enrichment, and role-targeted decision-maker
// discovery. Auth is header-only (x-api-key) — Apollo removed query/body
// param auth in Sept 2024, and OAuth on Apollo is partner-only, not
// available for regular API-key customers (docs.apollo.io/reference/authentication).
type ApolloClient struct {
	httpClient *http.Client
}

func NewApolloClient() *ApolloClient {
	return &ApolloClient{httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *ApolloClient) Provider() string { return "apollo" }

func apolloAuth(req *http.Request, credentials map[string]string) error {
	apiKey := strings.TrimSpace(credentials["api_key"])
	if apiKey == "" {
		return fmt.Errorf("api_key is required")
	}
	req.Header.Set("x-api-key", apiKey)
	return nil
}

// doJSON issues one Apollo API call and maps rate-limit/credit-exhaustion
// responses to RateLimitError so the worker backs off instead of hammering
// a provider that's already told us to slow down.
func (c *ApolloClient) doJSON(ctx context.Context, method, url string, credentials map[string]string, body interface{}) (map[string]interface{}, error) {
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	if err := apolloAuth(req, credentials); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apollo request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	// Apollo's docs don't fully enumerate error bodies for 402/429
	// (docs.apollo.io/reference/status-codes admits this), so we key off
	// status code alone rather than trying to parse an assumed shape.
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusPaymentRequired {
		return nil, &RateLimitError{Provider: "apollo", RetryAfter: resp.Header.Get("Retry-After")}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &AuthExpiredError{Provider: "apollo"}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("apollo API returned HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode apollo response: %w", err)
	}
	return out, nil
}

func (c *ApolloClient) Validate(ctx context.Context, credentials map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.apollo.io/v1/auth/health", nil)
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
	return nil
}

// EnrichOrganization pulls firmographic data for a domain: industry,
// employee count, revenue, funding, technologies, location. Returns
// (nil, nil) — not an error — when Apollo simply has no record for the
// domain, so callers don't treat "no data" as a sync failure.
func (c *ApolloClient) EnrichOrganization(ctx context.Context, credentials map[string]string, domain string) (map[string]interface{}, error) {
	url := "https://api.apollo.io/api/v1/organizations/enrich?domain=" + strings.TrimSpace(domain)
	result, err := c.doJSON(ctx, http.MethodGet, url, credentials, nil)
	if err != nil {
		return nil, err
	}
	org, _ := result["organization"].(map[string]interface{})
	if org == nil {
		return nil, nil
	}
	return org, nil
}

// TargetRoles is the fixed checklist of sponsorship-relevant roles
// SponsorOS attempts to find at every company. Order matters: earlier
// roles get enrichment priority when the per-run credit cap is hit.
var TargetRoles = []string{
	"Founder", "CEO", "Co-Founder", "CMO", "Marketing Director",
	"Head of Partnerships", "Partnership Manager", "Developer Relations",
	"Community Lead", "Brand Lead", "Country Manager", "Regional Manager",
	"Events Manager", "Communications Lead",
}

var apolloSeniorities = []string{"owner", "founder", "c_suite", "vp", "head", "director", "manager"}

// maxEmailRevealsPerCompany bounds how many people/match (credit-consuming)
// calls one DiscoverRoleContacts run makes, so a single company can't burn
// an unbounded number of credits in one sync pass.
const maxEmailRevealsPerCompany = 6

type apolloPerson struct {
	ID          string `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	LinkedinURL string `json:"linkedin_url"`
	Email       string `json:"email"`
	EmailStatus string `json:"email_status"`
}

// DiscoverRoleContacts searches for each role in TargetRoles at the given
// company and reports one DiscoveredContactRecord per role — Available
// false (never fabricated) when nobody matching that role turns up. Emails
// are revealed (credit-consuming) for only the first
// maxEmailRevealsPerCompany matches, prioritized in TargetRoles order.
func (c *ApolloClient) DiscoverRoleContacts(ctx context.Context, credentials map[string]string, companyName, domain string) ([]DiscoveredContactRecord, error) {
	if strings.TrimSpace(domain) == "" {
		return nil, fmt.Errorf("company has no domain to search by")
	}

	body := map[string]interface{}{
		"q_organization_domains_list": []string{domain},
		"person_titles":               TargetRoles,
		"include_similar_titles":      true,
		"person_seniorities":          apolloSeniorities,
		"page":                        1,
		"per_page":                    25,
	}

	result, err := c.doJSON(ctx, http.MethodPost, "https://api.apollo.io/api/v1/mixed_people/api_search", credentials, body)
	if err != nil {
		return nil, err
	}

	rawPeople, _ := result["people"].([]interface{})
	people := make([]apolloPerson, 0, len(rawPeople))
	for _, p := range rawPeople {
		b, _ := json.Marshal(p)
		var person apolloPerson
		if err := json.Unmarshal(b, &person); err == nil && person.ID != "" {
			people = append(people, person)
		}
	}

	assigned := make(map[string]bool) // person ID -> already matched to a role
	records := make([]DiscoveredContactRecord, 0, len(TargetRoles))
	reveals := 0

	for _, role := range TargetRoles {
		match := findBestRoleMatch(role, people, assigned)
		if match == nil {
			records = append(records, DiscoveredContactRecord{
				CompanyName:   companyName,
				CompanyDomain: domain,
				RoleQueried:   role,
				Available:     false,
				Source:        "apollo",
			})
			continue
		}
		assigned[match.ID] = true

		name := match.Name
		if name == "" {
			name = strings.TrimSpace(match.FirstName + " " + match.LastName)
		}

		record := DiscoveredContactRecord{
			CompanyName:   companyName,
			CompanyDomain: domain,
			RoleQueried:   role,
			Available:     true,
			Name:          name,
			Title:         match.Title,
			LinkedinURL:   match.LinkedinURL,
			Email:         match.Email,
			EmailStatus:   "unavailable",
			Source:        "apollo",
			Confidence:    0.5, // search-only match, no email confirmed yet
		}

		if reveals < maxEmailRevealsPerCompany {
			if enriched, err := c.revealEmail(ctx, credentials, match.ID); err == nil && enriched != nil {
				record.Email = enriched.Email
				record.EmailStatus = enriched.EmailStatus
				record.Confidence = confidenceFromEmailStatus(enriched.EmailStatus)
			}
			reveals++
		}

		records = append(records, record)
	}

	return records, nil
}

// revealEmail calls the credit-consuming people/match endpoint for one
// specific Apollo person ID to reveal their verified/guessed email.
func (c *ApolloClient) revealEmail(ctx context.Context, credentials map[string]string, personID string) (*apolloPerson, error) {
	body := map[string]interface{}{
		"id":                     personID,
		"reveal_personal_emails": true,
	}
	result, err := c.doJSON(ctx, http.MethodPost, "https://api.apollo.io/api/v1/people/match", credentials, body)
	if err != nil {
		return nil, err
	}
	person, _ := result["person"].(map[string]interface{})
	if person == nil {
		return nil, nil
	}
	b, _ := json.Marshal(person)
	var out apolloPerson
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func confidenceFromEmailStatus(status string) float64 {
	switch status {
	case "verified":
		return 0.95
	case "guessed":
		return 0.4
	case "unverified":
		return 0.3
	default:
		return 0.5
	}
}

// findBestRoleMatch finds the not-yet-assigned person whose title best
// matches the requested role (case-insensitive substring match either
// direction, since Apollo titles are free text like "VP of Partnerships"
// for a "Head of Partnerships" query).
func findBestRoleMatch(role string, people []apolloPerson, assigned map[string]bool) *apolloPerson {
	roleLower := strings.ToLower(role)
	roleWords := strings.Fields(roleLower)

	for i := range people {
		p := &people[i]
		if assigned[p.ID] || p.Title == "" {
			continue
		}
		titleLower := strings.ToLower(p.Title)
		if strings.Contains(titleLower, roleLower) || strings.Contains(roleLower, titleLower) {
			return p
		}
	}
	// Fall back to a keyword overlap match (e.g. "Head of Partnerships" vs "Partnerships Lead").
	for i := range people {
		p := &people[i]
		if assigned[p.ID] || p.Title == "" {
			continue
		}
		titleLower := strings.ToLower(p.Title)
		for _, w := range roleWords {
			if len(w) > 3 && strings.Contains(titleLower, w) {
				return p
			}
		}
	}
	return nil
}

func (c *ApolloClient) Sync(ctx context.Context, credentials map[string]string, _ map[string]interface{}) (*SyncResult, error) {
	body := map[string]interface{}{"page": 1, "per_page": 25}
	result, err := c.doJSON(ctx, http.MethodPost, "https://api.apollo.io/api/v1/mixed_people/api_search", credentials, body)
	if err != nil {
		return nil, err
	}

	var people []apolloPerson
	rawPeople, _ := result["people"].([]interface{})
	for _, p := range rawPeople {
		b, _ := json.Marshal(p)
		var person apolloPerson
		if err := json.Unmarshal(b, &person); err == nil {
			people = append(people, person)
		}
	}

	pagination, _ := result["pagination"].(map[string]interface{})
	totalEntries := 0
	if te, ok := pagination["total_entries"].(float64); ok {
		totalEntries = int(te)
	}

	contacts := make([]ContactRecord, 0, len(people))
	for _, p := range people {
		if p.FirstName == "" && p.LastName == "" && p.Name == "" {
			continue
		}
		first, last := p.FirstName, p.LastName
		if first == "" && last == "" {
			first, last = splitName(p.Name)
		}
		contacts = append(contacts, ContactRecord{
			FirstName:   first,
			LastName:    last,
			Email:       p.Email,
			Title:       p.Title,
			LinkedinURL: p.LinkedinURL,
		})
	}

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"contacts_found": len(people),
			"total_entries":  totalEntries,
		},
		Contacts: contacts,
	}, nil
}
