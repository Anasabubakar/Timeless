package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
	"github.com/timeless/backend/internal/worker"
)

type IntegrationService struct {
	repo    *repository.IntegrationRepository
	clients map[string]integration.Client
	cipher  *security.CredentialCipher
	worker  *worker.Client
}

func NewIntegrationService(repo *repository.IntegrationRepository, cipher *security.CredentialCipher, workerClient *worker.Client) *IntegrationService {
	return &IntegrationService{
		repo:    repo,
		clients: integration.Registry(),
		cipher:  cipher,
		worker:  workerClient,
	}
}

func (s *IntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]models.Integration, error) {
	return s.repo.List(ctx, orgID)
}

func (s *IntegrationService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Integration, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *IntegrationService) Create(ctx context.Context, integration *models.Integration) error {
	return s.repo.Create(ctx, integration)
}

func (s *IntegrationService) Update(ctx context.Context, integration *models.Integration) error {
	return s.repo.Update(ctx, integration)
}

func (s *IntegrationService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}

type ConnectInput struct {
	Credentials map[string]string `json:"credentials"`
}

// Connect validates real credentials against the provider, persists them
// encrypted, and kicks off a background sync job — the connection is live
// the moment this returns, not simulated.
func (s *IntegrationService) Connect(ctx context.Context, orgID, userID uuid.UUID, provider string, input ConnectInput) (*models.Integration, error) {
	client, ok := s.clients[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}

	if err := client.Validate(ctx, input.Credentials); err != nil {
		return nil, fmt.Errorf("connect %s: %w", provider, err)
	}

	encryptedCreds, err := s.encryptCredentials(input.Credentials)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	rec, err := s.repo.GetByProvider(ctx, orgID, provider)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		rec = &models.Integration{
			OrganizationID: orgID,
			Provider:       provider,
			Type:           providerType(provider),
			Name:           providerName(provider),
			InstalledBy:    &userID,
		}
	}

	rec.Status = "syncing"
	rec.Credentials = encryptedCreds
	rec.LastError = nil

	if rec.ID == uuid.Nil {
		if err := s.repo.Create(ctx, rec); err != nil {
			return nil, err
		}
	} else if err := s.repo.Update(ctx, rec); err != nil {
		return nil, err
	}

	if s.worker != nil {
		if _, err := s.worker.Enqueue(worker.TaskIntegrationSync, worker.TaskPayload{
			OrgID:      orgID.String(),
			UserID:     userID.String(),
			EntityID:   rec.ID.String(),
			EntityType: "integration",
			Action:     provider,
		}); err != nil {
			return nil, fmt.Errorf("enqueue sync job: %w", err)
		}
	}

	return rec, nil
}

func (s *IntegrationService) encryptCredentials(credentials map[string]string) (datatypes.JSON, error) {
	plain, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	enc, err := s.cipher.Encrypt(string(plain))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"enc": enc})
}

func providerType(provider string) string {
	if provider == "zapier" {
		return "zapier"
	}
	return "native"
}

func providerName(provider string) string {
	switch provider {
	case "zapier":
		return "Zapier"
	case "notion":
		return "Notion"
	case "apollo":
		return "Apollo"
	default:
		return provider
	}
}
