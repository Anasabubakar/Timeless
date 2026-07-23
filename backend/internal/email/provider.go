package email

import "context"

type Message struct {
	From        string
	FromName    string
	To          []string
	CC          []string
	BCC         []string
	ReplyTo     string
	Subject     string
	TextBody    string
	HTMLBody    string
	Headers     map[string]string
	Attachments []Attachment
	Tags        map[string]string
}

type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

type SendResult struct {
	MessageID  string
	ProviderID string
	Status     string
}

type Provider interface {
	Name() string
	Send(ctx context.Context, msg *Message) (*SendResult, error)
}
