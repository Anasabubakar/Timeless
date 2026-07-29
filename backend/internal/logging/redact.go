// Package logging provides a drop-in replacement for the standard
// library's log.Printf that scrubs known-sensitive patterns before a
// line ever reaches stdout/a log aggregator. Nothing in this codebase
// used a structured logger before this — every log.Printf call site
// formats whatever error/value it was handed directly, which is fine
// for a plain error string but a real risk the moment that string
// happens to embed a token, password, or API key (an upstream OAuth
// provider's error response, for example, can echo back parts of the
// request that included one).
package logging

import "regexp"

// redactionPatterns match the shapes secrets tend to take in error
// strings and log lines, independent of which field they came from —
// this is a safety net for values that end up embedded in free-form
// text (an error message, a URL, a header dump), not a substitute for
// not logging known-sensitive fields in the first place.
var fullMatchPatterns = []*regexp.Regexp{
	// Authorization: Bearer <token> / Basic <token>
	regexp.MustCompile(`(?i)(bearer|basic)\s+[A-Za-z0-9\-._~+/]{8,}=*`),
	// JWTs: three base64url segments separated by dots.
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\b`),
}

// keyValuePattern matches key="value" / key=value / key: value where
// key names a secret, across the common separator styles (JSON, query
// string, plain kv). The key name is kept in the replacement (via
// capture group $1) so a redacted line still says *which* secret it
// was, without ever printing the value itself.
var keyValuePattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|private[_-]?key)["']?\s*[:=]\s*["']?[^\s"'&,}]{4,}`)

const redactedPlaceholder = "[REDACTED]"

// Redact scans s for known secret shapes and replaces each match with a
// placeholder, preserving the rest of the message (and, for key=value
// pairs, the key name) so it's still useful for debugging.
func Redact(s string) string {
	for _, pattern := range fullMatchPatterns {
		s = pattern.ReplaceAllString(s, redactedPlaceholder)
	}
	s = keyValuePattern.ReplaceAllString(s, "$1="+redactedPlaceholder)
	return s
}
