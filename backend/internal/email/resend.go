package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Resend struct {
	apiKey string
	client *http.Client
}

func NewResend(apiKey string) *Resend {
	return &Resend{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (r *Resend) Name() string {
	return "resend"
}

func (r *Resend) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	payload := r.buildPayload(msg)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("resend: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("resend: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("resend: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(respBody, &result)

	return &SendResult{
		MessageID:  result.ID,
		ProviderID: "resend",
		Status:     "sent",
	}, nil
}

type resendPayload struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	CC          []string           `json:"cc,omitempty"`
	BCC         []string           `json:"bcc,omitempty"`
	ReplyTo     string             `json:"reply_to,omitempty"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	Tags        []resendTag        `json:"tags,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

type resendTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func (r *Resend) buildPayload(msg *Message) resendPayload {
	from := msg.From
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
	}

	p := resendPayload{
		From:    from,
		To:      msg.To,
		CC:      msg.CC,
		BCC:     msg.BCC,
		ReplyTo: msg.ReplyTo,
		Subject: msg.Subject,
		HTML:    msg.HTMLBody,
		Text:    msg.TextBody,
		Headers: msg.Headers,
	}

	// Resend tag names/values are restricted to ASCII letters, numbers,
	// underscores, and dashes — unlike Message.Tags (a free-form map,
	// e.g. every category value in this codebase is dot-separated:
	// "auth.verify_email", "team.invitation"). Sanitizing rather than
	// dropping keeps every send still categorized in Resend's dashboard
	// — "team.invitation" becomes "team_invitation" — instead of every
	// single email sent through this provider silently losing all of
	// its tags just because this app's tag convention uses dots.
	for k, v := range msg.Tags {
		if k == "" || v == "" {
			continue
		}
		p.Tags = append(p.Tags, resendTag{Name: sanitizeResendTag(k), Value: sanitizeResendTag(v)})
	}

	if len(msg.Attachments) > 0 {
		p.Attachments = make([]resendAttachment, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			p.Attachments = append(p.Attachments, resendAttachment{
				Filename: a.Filename,
				Content:  base64.StdEncoding.EncodeToString(a.Content),
			})
		}
	}

	return p
}

// sanitizeResendTag replaces every character outside Resend's allowed
// tag charset (ASCII letters, numbers, underscore, dash) with an
// underscore, so a dot-separated category like "team.invitation" still
// reads recognizably as "team_invitation" instead of being dropped.
func sanitizeResendTag(s string) string {
	b := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b = append(b, r)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}
