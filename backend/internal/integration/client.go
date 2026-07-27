// Package integration holds stateless connectors to third-party providers.
// Each call takes credentials explicitly (rather than holding connection
// state on the struct) so one process can serve many organizations
// concurrently without one org's credentials leaking into another's calls.
package integration

import "context"

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
	Title string
	URL   string
}

type SyncResult struct {
	Provider string                 `json:"provider"`
	Details  map[string]interface{} `json:"details"`
	Contacts []ContactRecord        `json:"-"`
	Notes    []NoteRecord           `json:"-"`
}

// Client validates and syncs data for one third-party provider.
type Client interface {
	Provider() string
	// Validate checks that the supplied credentials actually authenticate
	// against the provider. Returning nil means the connection is real and
	// live, not simulated.
	Validate(ctx context.Context, credentials map[string]string) error
	// Sync performs a lightweight pull to confirm the connection is usable
	// end-to-end and report back a coarse summary of what's available.
	Sync(ctx context.Context, credentials map[string]string) (*SyncResult, error)
}

// Registry returns every provider client Phase 1 supports, keyed by the
// `provider` value stored on models.Integration.
func Registry() map[string]Client {
	return map[string]Client{
		"zapier": NewZapierClient(),
		"notion": NewNotionClient(),
		"apollo": NewApolloClient(),
	}
}
