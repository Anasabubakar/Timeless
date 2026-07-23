package worker

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func uuidFromString(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

type EmailPayload struct {
	OrgID    string   `json:"org_id"`
	To       []string `json:"to"`
	CC       []string `json:"cc"`
	BCC      []string `json:"bcc"`
	Subject  string   `json:"subject"`
	TextBody string   `json:"text_body"`
	HTMLBody string   `json:"html_body"`
	ReplyTo  string   `json:"reply_to"`
	Provider string   `json:"provider,omitempty"`
}

func (c *Client) EnqueueEmail(payload EmailPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal email payload: %w", err)
	}
	task := asynq.NewTask("email:deliver", data)
	info, err := c.inner.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return "", fmt.Errorf("enqueue email: %w", err)
	}
	return info.ID, nil
}

func (c *Client) EnqueueMemoryIndex(payload MemoryIndexPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal memory index payload: %w", err)
	}
	task := asynq.NewTask("memory:index:v2", data)
	info, err := c.inner.Enqueue(task, asynq.Queue("low"))
	if err != nil {
		return "", fmt.Errorf("enqueue memory index: %w", err)
	}
	return info.ID, nil
}
