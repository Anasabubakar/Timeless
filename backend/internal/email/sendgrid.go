package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SendGrid struct {
	apiKey string
	client *http.Client
}

func NewSendGrid(apiKey string) *SendGrid {
	return &SendGrid{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (s *SendGrid) Name() string {
	return "sendgrid"
}

func (s *SendGrid) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	payload := s.buildPayload(msg)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("sendgrid: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sendgrid: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sendgrid: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sendgrid: API returned status %d", resp.StatusCode)
	}

	msgID := resp.Header.Get("X-Message-Id")

	return &SendResult{
		MessageID:  msgID,
		ProviderID: "sendgrid",
		Status:     "sent",
	}, nil
}

type sgPayload struct {
	Personalizations []sgPersonalization `json:"personalizations"`
	From             sgEmail             `json:"from"`
	ReplyTo          *sgEmail            `json:"reply_to,omitempty"`
	Subject          string              `json:"subject"`
	Content          []sgContent         `json:"content"`
}

type sgPersonalization struct {
	To  []sgEmail `json:"to"`
	CC  []sgEmail `json:"cc,omitempty"`
	BCC []sgEmail `json:"bcc,omitempty"`
}

type sgEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (s *SendGrid) buildPayload(msg *Message) sgPayload {
	tos := make([]sgEmail, 0, len(msg.To))
	for _, t := range msg.To {
		tos = append(tos, sgEmail{Email: t})
	}

	ccs := make([]sgEmail, 0, len(msg.CC))
	for _, c := range msg.CC {
		ccs = append(ccs, sgEmail{Email: c})
	}

	bccs := make([]sgEmail, 0, len(msg.BCC))
	for _, b := range msg.BCC {
		bccs = append(bccs, sgEmail{Email: b})
	}

	p := sgPayload{
		Personalizations: []sgPersonalization{{
			To:  tos,
			CC:  ccs,
			BCC: bccs,
		}},
		From:    sgEmail{Email: msg.From, Name: msg.FromName},
		Subject: msg.Subject,
	}

	if msg.ReplyTo != "" {
		p.ReplyTo = &sgEmail{Email: msg.ReplyTo}
	}

	if msg.HTMLBody != "" {
		p.Content = append(p.Content, sgContent{Type: "text/html", Value: msg.HTMLBody})
	} else if msg.TextBody != "" {
		p.Content = append(p.Content, sgContent{Type: "text/plain", Value: msg.TextBody})
	}

	return p
}
