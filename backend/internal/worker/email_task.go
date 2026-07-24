package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/timeless/backend/internal/email"
)

const TaskEmailDeliver = "email:deliver"

type EmailDeliveryHandler struct {
	sender *email.Sender
}

func NewEmailDeliveryHandler(sender *email.Sender) *EmailDeliveryHandler {
	return &EmailDeliveryHandler{sender: sender}
}

func (h *EmailDeliveryHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var payload EmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal email payload: %w", err)
	}

	msg := &email.Message{
		To:       payload.To,
		CC:       payload.CC,
		BCC:      payload.BCC,
		Subject:  payload.Subject,
		TextBody: payload.TextBody,
		HTMLBody: payload.HTMLBody,
		ReplyTo:  payload.ReplyTo,
		Tags:     map[string]string{"org_id": payload.OrgID},
	}

	var result *email.SendResult
	var err error

	if payload.Provider != "" {
		result, err = h.sender.SendWith(ctx, payload.Provider, msg)
	} else {
		result, err = h.sender.Send(ctx, msg)
	}

	if err != nil {
		return fmt.Errorf("email delivery failed: %w", err)
	}

	_ = result
	return nil
}

func RegisterEmailDeliveryHandler(mux *asynq.ServeMux, sender *email.Sender) {
	h := NewEmailDeliveryHandler(sender)
	mux.HandleFunc(TaskEmailDeliver, h.Handle)
}
