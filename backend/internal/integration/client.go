// Package integration holds stateless connectors to third-party providers.
// Each call takes credentials explicitly (rather than holding connection
// state on the struct) so one process can serve many organizations
// concurrently without one org's credentials leaking into another's calls.
package integration

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// ContactRecord is a lightweight, provider-agnostic shape for a person +
// their company, used to populate the CRM (Company/Contact tables) from
// whatever a provider's sync surfaces.
type ContactRecord struct {
	FirstName      string
	LastName       string
	Email          string
	Title          string
	LinkedinURL    string
	CompanyName    string
	CompanyDomain  string
	CompanyWebsite string
}

// NoteRecord is a lightweight reference to a document/page a provider
// surfaced, used to populate Recent Activity.
type NoteRecord struct {
	Title      string
	URL        string
	ExternalID string // provider's own id, so re-syncing can dedupe/update instead of re-creating
	UpdatedAt  string // RFC3339, provider's last-edited timestamp, for conflict-safe writes
}

// DiscoveredContactRecord is a role-targeted contact found for a specific
// company (e.g. "who is the Head of Partnerships at Acme?"). Unlike
// ContactRecord, it always carries how confident we are and where the data
// came from, and Available=false when the role search came back empty — we
// never invent a name/email/title to fill a gap.
type DiscoveredContactRecord struct {
	CompanyName   string
	CompanyDomain string
	RoleQueried   string // e.g. "Head of Partnerships"
	Available     bool
	Name          string
	Title         string
	Email         string
	EmailStatus   string // verified | guessed | unverified | unavailable
	LinkedinURL   string
	Confidence    float64 // 0..1, derived from email_status + match quality
	Source        string  // "apollo"
}

// SyncResult is what one Sync() pass reports back. State is merged (not
// replaced) into the integration's stored config by the caller, so a
// provider can carry forward cursors/watermarks (e.g. Notion's
// last-synced-at) across incremental runs without the caller needing to
// know what's inside it.
type SyncResult struct {
	Provider           string                    `json:"provider"`
	Details            map[string]interface{}    `json:"details"`
	State              map[string]interface{}    `json:"-"`
	Contacts           []ContactRecord           `json:"-"`
	Notes              []NoteRecord              `json:"-"`
	DiscoveredContacts []DiscoveredContactRecord `json:"-"`
	Warnings           []string                  `json:"-"`
}

// Client validates and syncs data for one third-party provider.
type Client interface {
	Provider() string
	// Validate checks that the supplied credentials actually authenticate
	// against the provider. Returning nil means the connection is real and
	// live, not simulated.
	Validate(ctx context.Context, credentials map[string]string) error
	// Sync pulls real data from the provider. state is the previous run's
	// SyncResult.State (nil on first connect), letting a provider resume
	// from a cursor/watermark instead of re-processing everything.
	Sync(ctx context.Context, credentials map[string]string, state map[string]interface{}) (*SyncResult, error)
}

// Refresher is implemented by providers whose OAuth tokens expire and can
// be silently renewed with a refresh token. The caller must persist the
// returned credentials (both fields may have rotated).
type Refresher interface {
	Refresh(ctx context.Context, credentials map[string]string) (map[string]string, error)
}

// RegistryConfig carries the provider-specific settings Registry() needs to
// construct clients (OAuth client id/secret for providers that support
// token refresh). Kept minimal and provider-agnostic rather than importing
// the whole app config package here.
type RegistryConfig struct {
	NotionClientID     string
	NotionClientSecret string
}

// Registry returns every provider client this app supports, keyed by the
// `provider` value stored on models.Integration.
func Registry(cfg RegistryConfig) map[string]Client {
	return map[string]Client{
		"zapier": NewZapierClient(),
		"notion": NewNotionClient(cfg.NotionClientID, cfg.NotionClientSecret),
		"apollo": NewApolloClient(),
	}
}

// AuthExpiredError signals that a provider rejected our credentials as
// expired or revoked (typically HTTP 401). Callers should attempt a
// Refresher.Refresh if the client supports it, and otherwise mark the
// integration as needing reconnect rather than retrying blindly.
type AuthExpiredError struct {
	Provider string
}

func (e *AuthExpiredError) Error() string {
	return fmt.Sprintf("%s authorization expired or was revoked — reconnect required", e.Provider)
}

// RateLimitError signals that a provider rejected a request for being over
// its rate limit (or, for Apollo, out of credits). The worker uses
// RetryAfterDuration to schedule the retry instead of guessing a backoff.
type RateLimitError struct {
	Provider   string
	RetryAfter string // raw Retry-After header value, if the provider sent one
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("%s rate limit hit, retry after %s", e.Provider, e.RetryAfter)
	}
	return fmt.Sprintf("%s rate limit hit", e.Provider)
}

// RetryAfterDuration parses the Retry-After header (seconds, per HTTP spec)
// and falls back to a conservative default when the provider didn't send one.
func (e *RateLimitError) RetryAfterDuration(fallback time.Duration) time.Duration {
	if e.RetryAfter == "" {
		return fallback
	}
	if secs, err := strconv.Atoi(e.RetryAfter); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return fallback
}
