package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
	"github.com/sponsoros/backend/internal/worker"
)

type WebhookService struct {
	repo   *repository.WebhookRepository
	client *worker.Client
}

func NewWebhookService(repo *repository.WebhookRepository, client *worker.Client) *WebhookService {
	return &WebhookService{repo: repo, client: client}
}

func (s *WebhookService) List(ctx context.Context, orgID uuid.UUID) ([]models.Webhook, error) {
	return s.repo.List(ctx, orgID)
}

func (s *WebhookService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Webhook, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *WebhookService) Create(ctx context.Context, webhook *models.Webhook) error {
	if webhook.Secret == "" {
		secret, err := generateWebhookSecret()
		if err != nil {
			return err
		}
		webhook.Secret = secret
	}
	return s.repo.Create(ctx, webhook)
}

func (s *WebhookService) Update(ctx context.Context, webhook *models.Webhook) error {
	return s.repo.Update(ctx, webhook)
}

func (s *WebhookService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}

func (s *WebhookService) RotateSecret(ctx context.Context, orgID, id uuid.UUID) (*models.Webhook, error) {
	webhook, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	secret, err := generateWebhookSecret()
	if err != nil {
		return nil, err
	}
	webhook.Secret = secret
	if err := s.repo.Update(ctx, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// TriggerEvent dispatches webhook deliveries for all active webhooks subscribed to the given event.
func (s *WebhookService) TriggerEvent(ctx context.Context, orgID uuid.UUID, event string, data map[string]interface{}) error {
	webhooks, err := s.repo.GetActiveByEvent(ctx, orgID, event)
	if err != nil {
		return err
	}

	for _, wh := range webhooks {
		payload := worker.TaskPayload{
			OrgID:      orgID.String(),
			EntityID:   wh.ID.String(),
			EntityType: "webhook",
			Action:     event,
			Data:       data,
		}
		if _, err := s.client.Enqueue(worker.TaskWebhookDeliver, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *WebhookService) ListDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit int) ([]models.WebhookDelivery, error) {
	return s.repo.ListDeliveries(ctx, orgID, webhookID, limit)
}

func (s *WebhookService) TestWebhook(ctx context.Context, orgID, webhookID uuid.UUID) error {
	webhook, err := s.repo.GetByID(ctx, orgID, webhookID)
	if err != nil {
		return err
	}

	testData := map[string]interface{}{
		"event":  "test.ping",
		"source": "sponsoros",
	}
	testPayload, _ := json.Marshal(testData)
	_ = testPayload

	payload := worker.TaskPayload{
		OrgID:      orgID.String(),
		EntityID:   webhook.ID.String(),
		EntityType: "webhook",
		Action:     "test.ping",
		Data:       testData,
	}
	_, err = s.client.Enqueue(worker.TaskWebhookDeliver, payload)
	return err
}

func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(b), nil
}
