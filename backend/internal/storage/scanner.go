package storage

import "context"

// Scanner checks file content for malware before it's persisted.
// Clean is false and Reason is set when the scan actively identified
// a threat; an error means the scan itself couldn't be completed
// (scanner unreachable, timeout, etc.) and callers should treat that as
// "don't trust this file" rather than silently allowing it through.
type Scanner interface {
	Scan(ctx context.Context, filename string, content []byte) (clean bool, reason string, err error)
}

// NoopScanner is the default: it allows everything. There's no
// malware-scanning infrastructure (ClamAV daemon, a cloud AV API, etc.)
// provisioned in this environment, so this exists to make the
// integration point explicit in code — a real Scanner (e.g. one
// dialing a clamd instance over TCP and speaking the INSTREAM protocol,
// or calling a hosted scanning API) is a drop-in replacement wherever
// NoopScanner is constructed, with no other code needing to change.
// This is documented as a known gap rather than silently absent — see
// the security report for the deployment-time follow-up.
type NoopScanner struct{}

func (NoopScanner) Scan(_ context.Context, _ string, _ []byte) (bool, string, error) {
	return true, "", nil
}
