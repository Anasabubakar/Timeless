package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	repo        *repository.IntegrationRepository
	syncRunRepo *repository.SyncRunRepository
	clients     map[string]integration.Client
	cipher      *security.CredentialCipher
	worker      *worker.Client
}

func NewIntegrationService(repo *repository.IntegrationRepository, syncRunRepo *repository.SyncRunRepository, cipher *security.CredentialCipher, workerClient *worker.Client) *IntegrationService {
	return &IntegrationService{
		repo:        repo,
		syncRunRepo: syncRunRepo,
		clients:     integration.Registry(),
		cipher:      cipher,
		worker:      workerClient,
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

// Revoke disconnects an integration without deleting its history: it wipes
// the stored credentials immediately (so a leaked DB row exposes nothing)
// and marks status "revoked" so the dashboard and reconnect flow can tell
// this apart from an integration that was simply never connected.
func (s *IntegrationService) Revoke(ctx context.Context, orgID, id uuid.UUID) error {
	rec, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return err
	}
	rec.Status = "revoked"
	rec.Credentials = datatypes.JSON([]byte(`{}`))
	return s.repo.Update(ctx, rec)
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
			Data:       map[string]interface{}{"trigger": "connect"},
		}); err != nil {
			return nil, fmt.Errorf("enqueue sync job: %w", err)
		}
	}

	return rec, nil
}

// DiscoverConnectedApps returns the third-party applications reachable
// through the org's Zapier connection (Google Calendar, Gmail, Slack,
// HubSpot, ...), used both for the "automatic discovery" requirement and to
// let other services check "does Zapier already cover this action?" before
// falling back to a native provider client.
func (s *IntegrationService) DiscoverConnectedApps(ctx context.Context, orgID uuid.UUID) ([]integration.ConnectedApp, bool, error) {
	rec, err := s.repo.GetByProvider(ctx, orgID, "zapier")
	if err != nil {
		return nil, false, fmt.Errorf("zapier is not connected")
	}
	credentials, err := s.decryptCredentials(rec.Credentials)
	if err != nil {
		return nil, false, err
	}
	zapier, ok := s.clients["zapier"].(*integration.ZapierClient)
	if !ok {
		return nil, false, fmt.Errorf("zapier client unavailable")
	}
	return zapier.DiscoverApps(ctx, credentials)
}

// ExecuteZapierAction is the entry point for "always try Zapier first":
// callers pass the action they want and its arguments, and this invokes it
// through the org's connected Zapier MCP server. Callers should fall back
// to a native client when this returns an error (e.g. zapier not connected,
// or the action isn't enabled for this user).
func (s *IntegrationService) ExecuteZapierAction(ctx context.Context, orgID uuid.UUID, action string, args map[string]interface{}) (map[string]interface{}, error) {
	rec, err := s.repo.GetByProvider(ctx, orgID, "zapier")
	if err != nil || rec.Status != "active" {
		return nil, fmt.Errorf("zapier is not connected")
	}
	credentials, err := s.decryptCredentials(rec.Credentials)
	if err != nil {
		return nil, err
	}
	zapier, ok := s.clients["zapier"].(*integration.ZapierClient)
	if !ok {
		return nil, fmt.Errorf("zapier client unavailable")
	}
	return zapier.ExecuteAction(ctx, credentials, action, args)
}

// DashboardEntry is one integration's health summary for the Integration
// Dashboard: connection status plus its recent sync history.
type DashboardEntry struct {
	Integration models.Integration `json:"integration"`
	RecentRuns  []models.SyncRun   `json:"recent_runs"`
	FailedRuns  int64              `json:"failed_runs_24h"`
	PendingJobs int                `json:"pending_jobs"`
}

// Dashboard aggregates connection health, sync history, and job status for
// every integration in the org, so the frontend never has to stitch this
// together from multiple round trips.
func (s *IntegrationService) Dashboard(ctx context.Context, orgID uuid.UUID) ([]DashboardEntry, error) {
	integrations, err := s.repo.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	entries := make([]DashboardEntry, 0, len(integrations))
	for _, in := range integrations {
		runs, err := s.syncRunRepo.ListByIntegration(ctx, orgID, in.ID, 20)
		if err != nil {
			return nil, err
		}

		var failed int64
		since := time.Now().Add(-24 * time.Hour)
		for _, r := range runs {
			if r.Status == "failed" && r.StartedAt.After(since) {
				failed++
			}
		}

		pending := 0
		if in.Status == "syncing" {
			pending = 1
		}

		entries = append(entries, DashboardEntry{
			Integration: in,
			RecentRuns:  runs,
			FailedRuns:  failed,
			PendingJobs: pending,
		})
	}
	return entries, nil
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

func (s *IntegrationService) decryptCredentials(raw datatypes.JSON) (map[string]string, error) {
	var stored struct {
		Enc string `json:"enc"`
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	plain, err := s.cipher.Decrypt(stored.Enc)
	if err != nil {
		return nil, err
	}
	var credentials map[string]string
	if err := json.Unmarshal([]byte(plain), &credentials); err != nil {
		return nil, err
	}
	return credentials, nil
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
