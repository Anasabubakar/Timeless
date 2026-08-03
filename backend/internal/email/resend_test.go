package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResendBuildPayload(t *testing.T) {
	r := NewResend("test-key")
	msg := &Message{
		From:     "noreply@timeless.app",
		FromName: "Timeless",
		To:       []string{"invitee@example.com"},
		Subject:  "You've been invited",
		HTMLBody: "<p>hi</p>",
		TextBody: "hi",
		Tags:     map[string]string{"category": "team.invitation"},
	}

	payload := r.buildPayload(msg)

	if payload.From != "Timeless <noreply@timeless.app>" {
		t.Errorf("From = %q, want %q", payload.From, "Timeless <noreply@timeless.app>")
	}
	if len(payload.To) != 1 || payload.To[0] != "invitee@example.com" {
		t.Errorf("To = %v", payload.To)
	}
	if len(payload.Tags) != 1 || payload.Tags[0].Name != "category" || payload.Tags[0].Value != "team_invitation" {
		t.Errorf("expected the dotted category value sanitized to team_invitation, got %v", payload.Tags)
	}
}

func TestResendSend(t *testing.T) {
	var receivedAuth string
	var receivedBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		receivedAuth = req.Header.Get("Authorization")
		_ = json.NewDecoder(req.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer srv.Close()

	r := NewResend("test-key")
	r.client = srv.Client()
	// resendAPIURL isn't a real field — Send() hardcodes the Resend
	// endpoint, so redirect via a transport-level override instead of a
	// URL field, keeping the production code free of test-only hooks.
	r.client.Transport = rewriteHostTransport{target: srv.URL}

	result, err := r.Send(context.Background(), &Message{
		From:     "noreply@timeless.app",
		To:       []string{"invitee@example.com"},
		Subject:  "Test",
		HTMLBody: "<p>hi</p>",
	})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if result.MessageID != "msg_123" {
		t.Errorf("MessageID = %q, want msg_123", result.MessageID)
	}
	if !strings.HasPrefix(receivedAuth, "Bearer test-key") {
		t.Errorf("Authorization header = %q, want Bearer test-key", receivedAuth)
	}
	if receivedBody["subject"] != "Test" {
		t.Errorf("subject = %v", receivedBody["subject"])
	}
}

// rewriteHostTransport redirects every request to target's host, so
// Send()'s hardcoded https://api.resend.com/emails URL can be tested
// against an httptest.Server without a production-code URL override.
type rewriteHostTransport struct {
	target string
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	targetURL, err := req.URL.Parse(t.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
	req.Host = targetURL.Host
	return http.DefaultTransport.RoundTrip(req)
}
