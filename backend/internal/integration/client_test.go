package integration

import (
	"errors"
	"testing"
	"time"
)

func TestRateLimitErrorRetryAfterDurationParsesSeconds(t *testing.T) {
	e := &RateLimitError{Provider: "notion", RetryAfter: "30"}
	if got := e.RetryAfterDuration(5 * time.Second); got != 30*time.Second {
		t.Errorf("RetryAfterDuration() = %v, want 30s", got)
	}
}

func TestRateLimitErrorRetryAfterDurationFallsBackWhenMissing(t *testing.T) {
	e := &RateLimitError{Provider: "apollo"}
	if got := e.RetryAfterDuration(15 * time.Second); got != 15*time.Second {
		t.Errorf("RetryAfterDuration() with no Retry-After = %v, want the 15s fallback", got)
	}
}

func TestRateLimitErrorRetryAfterDurationFallsBackOnGarbage(t *testing.T) {
	e := &RateLimitError{Provider: "apollo", RetryAfter: "not-a-number"}
	if got := e.RetryAfterDuration(10 * time.Second); got != 10*time.Second {
		t.Errorf("RetryAfterDuration() with unparseable header = %v, want the 10s fallback", got)
	}
}

func TestErrorTypesAreDistinguishableViaErrorsAs(t *testing.T) {
	var err error = &AuthExpiredError{Provider: "notion"}

	var authErr *AuthExpiredError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected errors.As to match *AuthExpiredError")
	}

	var rlErr *RateLimitError
	if errors.As(err, &rlErr) {
		t.Errorf("an AuthExpiredError should not also match *RateLimitError")
	}
}

func TestConflictErrorMessage(t *testing.T) {
	e := &ConflictError{Provider: "notion", Message: "page changed upstream"}
	if got := e.Error(); got != "notion conflict: page changed upstream" {
		t.Errorf("ConflictError.Error() = %q, want %q", got, "notion conflict: page changed upstream")
	}
}
