// Package normalize provides the canonicalization rules every integration
// ingestion path shares, so "Example.com", "www.example.com", and
// "https://example.com/" all resolve to the same company instead of
// silently creating three duplicates.
package normalize

import "strings"

// Email lowercases and trims — email addresses are case-insensitive at the
// domain part and conventionally treated so at the local part too for
// dedup purposes (providers rarely distinguish "Name@x.com" from
// "name@x.com" as different people).
func Email(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Domain strips scheme, "www.", path, port, and trailing slash so
// "https://www.Example.com/careers" and "example.com" compare equal.
func Domain(raw string) string {
	d := strings.ToLower(strings.TrimSpace(raw))
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "www.")
	if idx := strings.IndexAny(d, "/?#"); idx >= 0 {
		d = d[:idx]
	}
	if idx := strings.Index(d, ":"); idx >= 0 { // strip a port, e.g. example.com:8080
		d = d[:idx]
	}
	return strings.TrimSuffix(d, ".")
}

var companySuffixes = []string{
	" inc", " inc.", " incorporated",
	" llc", " llc.",
	" ltd", " ltd.", " limited",
	" co", " co.", " company",
	" corp", " corp.", " corporation",
	" gmbh", " plc", " lp", " llp",
}

// CompanyName lowercases, trims, and strips common legal suffixes so
// "Acme Inc." and "Acme" compare equal for dedup matching. This is only
// ever used as a comparison key — the original, human-entered name is
// always what's displayed/stored as the record of truth.
func CompanyName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.Join(strings.Fields(n), " ") // collapse internal whitespace
	for _, suffix := range companySuffixes {
		if strings.HasSuffix(n, suffix) {
			n = strings.TrimSuffix(n, suffix)
			break
		}
	}
	return strings.TrimSpace(n)
}

// PersonName trims and collapses whitespace for consistent first/last name
// storage regardless of how a provider formatted it.
func PersonName(name string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
}
