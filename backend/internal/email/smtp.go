package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/google/uuid"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	FromAddr string
	FromName string
	UseTLS   bool
}

type SMTP struct {
	cfg SMTPConfig
}

func NewSMTP(cfg SMTPConfig) *SMTP {
	return &SMTP{cfg: cfg}
}

func (s *SMTP) Name() string {
	return "smtp"
}

func (s *SMTP) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	from := msg.From
	if from == "" {
		from = s.cfg.FromAddr
	}

	allRecipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	allRecipients = append(allRecipients, msg.To...)
	allRecipients = append(allRecipients, msg.CC...)
	allRecipients = append(allRecipients, msg.BCC...)

	if len(allRecipients) == 0 {
		return nil, fmt.Errorf("smtp: no recipients")
	}

	body := s.buildMessage(msg, from)

	var conn net.Conn
	var err error

	if s.cfg.UseTLS {
		tlsCfg := &tls.Config{ServerName: s.cfg.Host}
		conn, err = tls.Dial("tcp", addr, tlsCfg)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("smtp: dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	if !s.cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{ServerName: s.cfg.Host}
			if err := client.StartTLS(tlsCfg); err != nil {
				return nil, fmt.Errorf("smtp: starttls: %w", err)
			}
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return nil, fmt.Errorf("smtp: auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return nil, fmt.Errorf("smtp: mail from: %w", err)
	}

	for _, rcpt := range allRecipients {
		if err := client.Rcpt(rcpt); err != nil {
			return nil, fmt.Errorf("smtp: rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return nil, fmt.Errorf("smtp: data: %w", err)
	}

	if _, err := w.Write([]byte(body)); err != nil {
		return nil, fmt.Errorf("smtp: write body: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("smtp: close data: %w", err)
	}

	client.Quit()

	msgID := uuid.New().String()

	return &SendResult{
		MessageID:  msgID,
		ProviderID: "smtp",
		Status:     "sent",
	}, nil
}

func (s *SMTP) buildMessage(msg *Message, from string) string {
	var b strings.Builder

	fromName := msg.FromName
	if fromName == "" {
		fromName = s.cfg.FromName
	}

	if fromName != "" {
		b.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	} else {
		b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}

	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))

	if len(msg.CC) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}

	if msg.ReplyTo != "" {
		b.WriteString(fmt.Sprintf("Reply-To: %s\r\n", msg.ReplyTo))
	}

	b.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	b.WriteString(fmt.Sprintf("Message-ID: <%s@%s>\r\n", uuid.New().String(), s.cfg.Host))

	for k, v := range msg.Headers {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	if msg.HTMLBody != "" {
		b.WriteString("MIME-Version: 1.0\r\n")
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.HTMLBody)
	} else {
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.TextBody)
	}

	return b.String()
}
