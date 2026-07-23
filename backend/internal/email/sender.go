package email

import (
	"context"
	"fmt"
	"html/template"
	"strings"
)

type Sender struct {
	registry *Registry
}

func NewSender(registry *Registry) *Sender {
	return &Sender{registry: registry}
}

func (s *Sender) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	provider, err := s.registry.Default()
	if err != nil {
		return nil, err
	}
	return provider.Send(ctx, msg)
}

func (s *Sender) SendWith(ctx context.Context, providerName string, msg *Message) (*SendResult, error) {
	provider, err := s.registry.Get(providerName)
	if err != nil {
		return nil, err
	}
	return provider.Send(ctx, msg)
}

type TemplateData struct {
	RecipientName  string
	RecipientEmail string
	SenderName     string
	OrgName        string
	Subject        string
	Variables      map[string]string
}

func (s *Sender) SendTemplate(ctx context.Context, msg *Message, tmplBody string, data TemplateData) (*SendResult, error) {
	rendered, err := renderTemplate(tmplBody, data)
	if err != nil {
		return nil, fmt.Errorf("email: render template: %w", err)
	}

	msg.HTMLBody = rendered
	return s.Send(ctx, msg)
}

func renderTemplate(tmplBody string, data TemplateData) (string, error) {
	t, err := template.New("email").Parse(tmplBody)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
