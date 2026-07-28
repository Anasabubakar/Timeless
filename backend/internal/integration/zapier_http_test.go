package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestReadMCPResponseHandlesPayloadOverDefaultScannerLimit is a regression
// test for a real failure found against a live Zapier MCP server: a
// workspace with many connected apps returns a tools/list payload over
// bufio.Scanner's 64KB default token size, which used to fail every sync
// with "bufio.Scanner: token too long".
func TestReadMCPResponseHandlesPayloadOverDefaultScannerLimit(t *testing.T) {
	bigPayload := `{"result":{"tools":[` + strings.Repeat(`{"name":"x"},`, 10000) + `{"name":"y"}]}}`
	if len(bigPayload) < 65536 {
		t.Fatalf("test payload isn't actually bigger than the 64KB default scanner limit")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: " + bigPayload + "\n"))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	payload, err := readMCPResponse(resp)
	if err != nil {
		t.Fatalf("readMCPResponse failed on an oversized event-stream line: %v", err)
	}
	if !strings.Contains(string(payload), `"name":"y"`) {
		t.Errorf("expected the full payload to be read, got a truncated/empty result")
	}
}

func TestReadMCPResponsePlainJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":{"tools":[]}}`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	payload, err := readMCPResponse(resp)
	if err != nil {
		t.Fatalf("readMCPResponse failed on a plain JSON body: %v", err)
	}
	if !strings.Contains(string(payload), `"tools"`) {
		t.Errorf("expected the plain JSON body to be returned as-is, got %s", payload)
	}
}

func TestZapierRPCMapsTooManyRequestsToRateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewZapierClient()
	_, err := c.rpc(context.Background(), srv.URL, "token", "tools/list", map[string]interface{}{})

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError for a 429 response, got %v", err)
	}
	if rlErr.RetryAfter != "5" {
		t.Errorf("expected Retry-After %q, got %q", "5", rlErr.RetryAfter)
	}
}
