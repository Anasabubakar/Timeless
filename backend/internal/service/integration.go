package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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
	repo          *repository.IntegrationRepository
	syncRunRepo   *repository.SyncRunRepository
	clients       map[string]integration.Client
	cipher        *security.CredentialCipher
	worker        *worker.Client
	publicBaseURL string
}

func NewIntegrationService(repo *repository.IntegrationRepository, syncRunRepo *repository.SyncRunRepository, cipher *security.CredentialCipher, workerClient *worker.Client, registryCfg integration.RegistryConfig, publicBaseURL string) *IntegrationService {
	return &IntegrationService{
		repo:          repo,
		syncRunRepo:   syncRunRepo,
		clients:       integration.Registry(registryCfg),
		cipher:        cipher,
		worker:        workerClient,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
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

// TriggerSync manually enqueues a sync for an already-connected integration
// — "sync now" instead of waiting for the next webhook/scheduled poll.
// Returns an error if credentials were never actually stored (e.g. the
// integration row exists but was created via the generic CRUD Create
// rather than a real Connect).
func (s *IntegrationService) TriggerSync(ctx context.Context, orgID, userID, id uuid.UUID) error {
	rec, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return err
	}
	if len(rec.Credentials) == 0 {
		return fmt.Errorf("integration has no stored credentials to sync with")
	}
	if s.worker == nil {
		return fmt.Errorf("worker client unavailable")
	}
	_, err = s.worker.Enqueue(worker.TaskIntegrationSync, worker.TaskPayload{
		OrgID:      orgID.String(),
		UserID:     userID.String(),
		EntityID:   rec.ID.String(),
		EntityType: "integration",
		Action:     rec.Provider,
		Data:       map[string]interface{}{"trigger": "manual"},
	})
	return err
}

// Revoke disconnects an integration without deleting its history: it wipes
// the stored credentials immediately (so a leaked DB row exposes nothing)
// and marks status "revoked" so the dashboard and reconnect flow can tell
// this apart from an integration that was simply never connected.
//
// This only revokes Timeless's own copy of the credentials — as of this
// writing, neither Notion's nor Apollo's OAuth API documents a token
// revocation endpoint (verified via their current docs), so there's no
// server-to-server call this can make to invalidate the token at the
// provider itself. The token stops being usable *by Timeless* the
// instant this runs; whether it's still technically valid at the
// provider until it naturally expires depends on the provider. Returns
// the provider name so the caller can surface that distinction instead
// of implying a stronger guarantee than this actually provides.
func (s *IntegrationService) Revoke(ctx context.Context, orgID, id uuid.UUID) (provider string, err error) {
	rec, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return "", err
	}
	rec.Status = "revoked"
	rec.Credentials = datatypes.JSON([]byte(`{}`))
	if err := s.repo.Update(ctx, rec); err != nil {
		return "", err
	}
	return rec.Provider, nil
}

// RotateCredentialsResult reports how many of the org's stored credentials
// were re-encrypted under the current key during a rotation pass.
type RotateCredentialsResult struct {
	Checked int `json:"checked"`
	Rotated int `json:"rotated"`
}

// RotateCredentials re-encrypts every integration whose stored credentials
// aren't already under the cipher's current key. Run this after deploying
// a new CREDENTIALS_ENCRYPTION_KEY (with the old key added to
// CREDENTIALS_ENCRYPTION_KEY_PREVIOUS so this pass can still decrypt the
// old rows) to finish the rotation instead of leaving old rows on the
// retired key indefinitely.
func (s *IntegrationService) RotateCredentials(ctx context.Context, orgID uuid.UUID) (*RotateCredentialsResult, error) {
	integrations, err := s.repo.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := &RotateCredentialsResult{}
	for i := range integrations {
		rec := &integrations[i]
		result.Checked++

		var stored struct {
			Enc string `json:"enc"`
		}
		if err := json.Unmarshal(rec.Credentials, &stored); err != nil || stored.Enc == "" {
			continue
		}
		if !s.cipher.NeedsRotation(stored.Enc) {
			continue
		}

		credentials, err := s.decryptCredentials(rec.Credentials)
		if err != nil {
			return result, fmt.Errorf("decrypt %s credentials: %w", rec.Provider, err)
		}
		encrypted, err := s.encryptCredentials(credentials)
		if err != nil {
			return result, fmt.Errorf("re-encrypt %s credentials: %w", rec.Provider, err)
		}
		rec.Credentials = encrypted
		if err := s.repo.Update(ctx, rec); err != nil {
			return result, err
		}
		result.Rotated++
	}
	return result, nil
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
	// workspace_id isn't a secret and isn't inside the encrypted blob on
	// its own — stash it in the clear so inbound webhooks (which carry a
	// workspace id but no org context of ours) can route back here.
	if wsID := input.Credentials["workspace_id"]; wsID != "" {
		rec.ExternalAccountID = &wsID
	}

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

// EnqueueWebhookSync routes an inbound provider webhook event to the right
// integration (matched by the provider's own account/workspace id, since
// the webhook request itself carries no org context of ours) and enqueues
// an incremental sync. Returns the matched org/integration id for logging.
func (s *IntegrationService) EnqueueWebhookSync(ctx context.Context, provider, externalAccountID string) (uuid.UUID, uuid.UUID, error) {
	rec, err := s.repo.GetByExternalAccountID(ctx, provider, externalAccountID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("no integration found for %s account %s: %w", provider, externalAccountID, err)
	}
	if s.worker == nil {
		return rec.OrganizationID, rec.ID, fmt.Errorf("worker client unavailable")
	}
	_, err = s.worker.Enqueue(worker.TaskIntegrationSync, worker.TaskPayload{
		OrgID:      rec.OrganizationID.String(),
		EntityID:   rec.ID.String(),
		EntityType: "integration",
		Action:     provider,
		Data:       map[string]interface{}{"trigger": "webhook"},
	})
	return rec.OrganizationID, rec.ID, err
}

// EnsureInboundWebhookToken returns the org's inbound webhook URL for
// provider, generating an unguessable token on first call and persisting
// it (idempotent after that — calling this again just returns the same
// URL). This is the auth mechanism for providers with no signing scheme
// of their own (Zapier's inbound "Webhooks by Zapier" trigger): the URL
// path segment IS the secret, so it must never be logged or exposed
// outside the authenticated dashboard call that returns it.
func (s *IntegrationService) EnsureInboundWebhookToken(ctx context.Context, orgID uuid.UUID, provider string) (*models.Integration, error) {
	rec, err := s.repo.GetByProvider(ctx, orgID, provider)
	if err != nil {
		return nil, fmt.Errorf("no %s integration connected for this organization", provider)
	}
	if rec.WebhookSecret != nil && *rec.WebhookSecret != "" {
		return rec, nil
	}

	token, err := generateWebhookToken()
	if err != nil {
		return nil, fmt.Errorf("generate webhook token: %w", err)
	}
	url := fmt.Sprintf("%s/api/v1/webhooks/%s/%s", s.publicBaseURL, provider, token)
	rec.WebhookSecret = &token
	rec.WebhookURL = &url
	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// IntegrationByWebhookToken resolves an inbound webhook request's org by
// its unguessable URL token — the sole authentication for providers with
// no signing mechanism of their own (see EnsureInboundWebhookToken). Only
// an active integration's token is honored, so revoking/disconnecting
// immediately stops inbound delivery from being accepted.
func (s *IntegrationService) IntegrationByWebhookToken(ctx context.Context, provider, token string) (*models.Integration, error) {
	if token == "" {
		return nil, fmt.Errorf("empty webhook token")
	}
	rec, err := s.repo.GetByWebhookSecret(ctx, provider, token)
	if err != nil {
		return nil, err
	}
	if rec.Status != "active" {
		return nil, fmt.Errorf("integration is not active")
	}
	return rec, nil
}

func generateWebhookToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// PushToNotionPage writes SponsorOS-side changes back to a Notion page,
// refusing (returning integration.ConflictError) rather than clobbering it
// if the page changed in Notion since expectedLastEditedTime.
func (s *IntegrationService) PushToNotionPage(ctx context.Context, orgID uuid.UUID, pageID string, properties map[string]interface{}, expectedLastEditedTime string) error {
	rec, err := s.repo.GetByProvider(ctx, orgID, "notion")
	if err != nil || rec.Status != "active" {
		return fmt.Errorf("notion is not connected")
	}
	credentials, err := s.decryptCredentials(rec.Credentials)
	if err != nil {
		return err
	}
	notion, ok := s.clients["notion"].(*integration.NotionClient)
	if !ok {
		return fmt.Errorf("notion client unavailable")
	}
	return notion.UpdatePageProperties(ctx, credentials, pageID, properties, expectedLastEditedTime)
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
