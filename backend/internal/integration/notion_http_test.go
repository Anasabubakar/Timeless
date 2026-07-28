package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotionValidateMapsUnauthorizedToAuthExpiredError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}`))
	}))
	defer srv.Close()

	c := NewNotionClient("", "")
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "expired-token", nil)

	var authErr *AuthExpiredError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthExpiredError for a 401 response, got %v", err)
	}
}

func TestNotionValidateMapsTooManyRequestsToRateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewNotionClient("", "")
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "token", nil)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError for a 429 response, got %v", err)
	}
	if rlErr.RetryAfter != "12" {
		t.Errorf("expected Retry-After to be captured as %q, got %q", "12", rlErr.RetryAfter)
	}
}

func TestNotionDoJSONSucceedsOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"user","id":"abc"}`))
	}))
	defer srv.Close()

	c := NewNotionClient("", "")
	result, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "token", nil)
	if err != nil {
		t.Fatalf("expected a successful 200 response to decode cleanly, got error: %v", err)
	}
	if result["id"] != "abc" {
		t.Errorf("decoded result = %v, want id=abc", result)
	}
}
