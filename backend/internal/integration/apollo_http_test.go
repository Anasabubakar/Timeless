package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApolloDoJSONMapsUnauthorizedToAuthExpiredError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewApolloClient()
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, map[string]string{"api_key": "bad-key"}, nil)

	var authErr *AuthExpiredError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthExpiredError for a 401 response, got %v", err)
	}
}

func TestApolloDoJSONMapsPaymentRequiredToRateLimitError(t *testing.T) {
	// Apollo's docs don't document a 402 body shape for out-of-credits, but
	// the status code itself must still be treated as retryable, not a
	// hard failure.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer srv.Close()

	c := NewApolloClient()
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, map[string]string{"api_key": "key"}, nil)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError for a 402 (out of credits) response, got %v", err)
	}
}

func TestApolloDoJSONRequiresAPIKey(t *testing.T) {
	c := NewApolloClient()
	_, err := c.doJSON(context.Background(), http.MethodGet, "http://example.invalid", map[string]string{}, nil)
	if err == nil {
		t.Fatalf("expected an error when no api_key is supplied")
	}
}

func TestApolloEnrichOrganizationReturnsNilWithoutError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`)) // Apollo has no record for this domain
	}))
	defer srv.Close()

	c := NewApolloClient()
	result, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, map[string]string{"api_key": "key"}, nil)
	if err != nil {
		t.Fatalf("expected no error for an empty-but-valid response, got %v", err)
	}
	if _, ok := result["organization"]; ok {
		t.Errorf("expected no organization key in an empty response, got %v", result)
	}
}
