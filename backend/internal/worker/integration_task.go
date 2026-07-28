package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
)

// integrationSyncRunner decrypts a connected integration's stored
// credentials and performs a real sync call against the provider. Every
// run is recorded in sync_runs (via syncRunRepo) so the dashboard has real
// history, and auth/rate-limit failures are handled distinctly: expired
// auth tries a token refresh before giving up, rate limits let asynq's
// retry mechanism back off using the provider's own Retry-After.
type integrationSyncRunner struct {
	db          *gorm.DB
	cipher      *security.CredentialCipher
	syncRunRepo *repository.SyncRunRepository
	clients     map[string]integration.Client
}

func newIntegrationSyncRunner(db *gorm.DB, cipher *security.CredentialCipher, syncRunRepo *repository.SyncRunRepository, registryCfg integration.RegistryConfig) *integrationSyncRunner {
	return &integrationSyncRunner{
		db:          db,
		cipher:      cipher,
		syncRunRepo: syncRunRepo,
		clients:     integration.Registry(registryCfg),
	}
}

func (r *integrationSyncRunner) run(ctx context.Context, integrationID, trigger string) error {
	var rec models.Integration
	if err := r.db.WithContext(ctx).Where("id = ?", integrationID).First(&rec).Error; err != nil {
		return fmt.Errorf("find integration: %w", err)
	}

	// A webhook-triggered sync can race a scheduled one for the same
	// connection; skip rather than double-process the same data.
	if running, err := r.syncRunRepo.HasRunning(ctx, rec.ID); err == nil && running {
		return nil
	}

	attempt := 1
	if n, ok := asynq.GetRetryCount(ctx); ok {
		attempt = n + 1
	}
	run, _ := r.syncRunRepo.Start(ctx, rec.OrganizationID, rec.ID, rec.Provider, trigger, attempt)

	client, ok := r.clients[rec.Provider]
	if !ok {
		err := fmt.Errorf("unsupported provider %q", rec.Provider)
		r.finishRun(ctx, run, "failed", 0, nil, err)
		return r.fail(ctx, &rec, err)
	}

	credentials, err := r.decryptCredentials(rec.Credentials)
	if err != nil {
		wrapped := fmt.Errorf("decrypt credentials: %w", err)
		r.finishRun(ctx, run, "failed", 0, nil, wrapped)
		return r.fail(ctx, &rec, wrapped)
	}

	state := extractState(rec.Config)
	result, syncErr := client.Sync(ctx, credentials, state)

	var authErr *integration.AuthExpiredError
	if errors.As(syncErr, &authErr) {
		if fresh, refreshErr := r.tryRefresh(ctx, client, &rec, credentials); refreshErr == nil {
			credentials = fresh
			result, syncErr = client.Sync(ctx, credentials, state)
		}
	}

	if syncErr != nil {
		if errors.As(syncErr, &authErr) {
			r.db.WithContext(ctx).Model(&rec).Updates(map[string]interface{}{
				"status": "expired", "last_error": syncErr.Error(),
			})
			r.finishRun(ctx, run, "failed", 0, nil, syncErr)
			// No point letting asynq retry — without a fresh reconnect the
			// next attempt will fail the exact same way.
			return fmt.Errorf("%s: %w", syncErr.Error(), asynq.SkipRetry)
		}

		var rlErr *integration.RateLimitError
		status := "error"
		if errors.As(syncErr, &rlErr) {
			status = "retrying"
		}
		r.db.WithContext(ctx).Model(&rec).Updates(map[string]interface{}{
			"status": status, "last_error": syncErr.Error(),
		})
		r.finishRun(ctx, run, "failed", 0, nil, syncErr)
		return syncErr
	}

	if err := r.ingestContacts(ctx, rec.OrganizationID, result.Contacts); err != nil {
		wrapped := fmt.Errorf("ingest contacts: %w", err)
		r.finishRun(ctx, run, "failed", 0, result.Warnings, wrapped)
		return r.fail(ctx, &rec, wrapped)
	}
	if err := r.ingestNotes(ctx, rec.OrganizationID, rec.Provider, result.Notes); err != nil {
		wrapped := fmt.Errorf("ingest notes: %w", err)
		r.finishRun(ctx, run, "failed", 0, result.Warnings, wrapped)
		return r.fail(ctx, &rec, wrapped)
	}
	if len(result.DiscoveredContacts) > 0 {
		if err := r.ingestDiscoveredContacts(ctx, rec.OrganizationID, result.DiscoveredContacts); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ingest discovered contacts: %v", err))
		}
	}

	recordsSynced := len(result.Contacts) + len(result.Notes) + len(result.DiscoveredContacts)

	// Apollo needs a dedicated per-company pass (org enrichment + role
	// discovery), which requires reading the org's existing companies —
	// something the generic, credential-only Client.Sync interface can't do.
	if rec.Provider == "apollo" {
		n, warnings := r.runApolloCompanyEnrichment(ctx, rec.OrganizationID, credentials)
		recordsSynced += n
		result.Warnings = append(result.Warnings, warnings...)
	}

	newConfig := mergeConfig(rec.Config, result.Details, result.State)
	now := time.Now()
	r.db.WithContext(ctx).Model(&rec).Updates(map[string]interface{}{
		"status":       "active",
		"last_sync_at": now,
		"last_error":   nil,
		"config":       newConfig,
	})

	r.finishRun(ctx, run, "success", recordsSynced, result.Warnings, nil)
	return nil
}

// tryRefresh attempts to silently renew expired credentials via the
// provider's Refresher implementation (if any) and persists the rotated
// credentials immediately so a second consecutive expiry isn't retrying
// with the same stale token.
func (r *integrationSyncRunner) tryRefresh(ctx context.Context, client integration.Client, rec *models.Integration, credentials map[string]string) (map[string]string, error) {
	refresher, ok := client.(integration.Refresher)
	if !ok {
		return nil, fmt.Errorf("provider %q does not support token refresh", rec.Provider)
	}
	fresh, err := refresher.Refresh(ctx, credentials)
	if err != nil {
		return nil, err
	}
	encrypted, err := r.encryptCredentials(fresh)
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(rec).Update("credentials", encrypted).Error; err != nil {
		return nil, err
	}
	rec.Credentials = encrypted
	return fresh, nil
}

// runApolloCompanyEnrichment enriches a bounded batch of the org's
// domain-having companies each run (oldest-updated first), so a workspace
// with many companies gets full coverage over successive syncs instead of
// front-loading all API/credit usage into one run.
func (r *integrationSyncRunner) runApolloCompanyEnrichment(ctx context.Context, orgID uuid.UUID, credentials map[string]string) (int, []string) {
	apolloClient, ok := r.clients["apollo"].(*integration.ApolloClient)
	if !ok {
		return 0, nil
	}

	var companies []models.Company
	r.db.WithContext(ctx).
		Where("organization_id = ? AND domain IS NOT NULL AND domain != ''", orgID).
		Order("updated_at ASC").
		Limit(10).
		Find(&companies)

	warnings := make([]string, 0)
	synced := 0

	for _, company := range companies {
		domain := ""
		if company.Domain != nil {
			domain = *company.Domain
		}
		if domain == "" {
			continue
		}

		if org, err := apolloClient.EnrichOrganization(ctx, credentials, domain); err != nil {
			warnings = append(warnings, fmt.Sprintf("apollo org enrich %s: %v", domain, err))
		} else if org != nil {
			enrichmentJSON, _ := json.Marshal(org)
			r.db.WithContext(ctx).Model(&company).Update("enrichment_data", datatypes.JSON(enrichmentJSON))
			synced++
		}

		records, err := apolloClient.DiscoverRoleContacts(ctx, credentials, company.Name, domain)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("apollo role discovery %s: %v", domain, err))
			continue
		}
		if err := r.ingestDiscoveredContacts(ctx, orgID, records); err != nil {
			warnings = append(warnings, fmt.Sprintf("ingest apollo contacts %s: %v", domain, err))
		}
		synced += len(records)
	}

	return synced, warnings
}

// ingestDiscoveredContacts upserts each available role match as a
// DecisionMaker (one per company+title), tagging confidence/source/email
// status in ProfileData so the UI can show provenance. Unavailable roles
// are never stored as rows — they're surfaced via sync warnings/details
// instead, since a missing row is the honest representation of "not found".
func (r *integrationSyncRunner) ingestDiscoveredContacts(ctx context.Context, orgID uuid.UUID, records []integration.DiscoveredContactRecord) error {
	for _, dc := range records {
		if !dc.Available {
			continue
		}

		var company models.Company
		q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
		if dc.CompanyDomain != "" {
			q = q.Where("domain = ?", dc.CompanyDomain)
		} else {
			q = q.Where("name = ?", dc.CompanyName)
		}
		if err := q.First(&company).Error; err != nil {
			continue
		}

		first, last := splitFullName(dc.Name)
		profileData, _ := json.Marshal(map[string]interface{}{
			"role_queried": dc.RoleQueried,
			"confidence":   dc.Confidence,
			"email_status": dc.EmailStatus,
			"source":       dc.Source,
		})

		var existing models.DecisionMaker
		err := r.db.WithContext(ctx).
			Where("organization_id = ? AND company_id = ? AND title = ?", orgID, company.ID, dc.Title).
			First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			dm := models.DecisionMaker{
				OrganizationID: orgID,
				CompanyID:      company.ID,
				FirstName:      first,
				LastName:       last,
				ProfileData:    profileData,
			}
			if dc.Title != "" {
				dm.Title = &dc.Title
			}
			if dc.Email != "" {
				dm.Email = &dc.Email
			}
			if dc.LinkedinURL != "" {
				dm.LinkedinURL = &dc.LinkedinURL
			}
			if err := r.db.WithContext(ctx).Create(&dm).Error; err != nil {
				return err
			}
		} else if err == nil {
			updates := map[string]interface{}{"profile_data": profileData}
			if dc.Email != "" {
				updates["email"] = dc.Email
			}
			if dc.LinkedinURL != "" {
				updates["linkedin_url"] = dc.LinkedinURL
			}
			if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func splitFullName(full string) (first, last string) {
	parts := make([]string, 0, 2)
	word := ""
	for _, r := range full + " " {
		if r == ' ' {
			if word != "" {
				parts = append(parts, word)
				word = ""
			}
			continue
		}
		word += string(r)
	}
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	last = parts[len(parts)-1]
	first = ""
	for i, p := range parts[:len(parts)-1] {
		if i > 0 {
			first += " "
		}
		first += p
	}
	return first, last
}

// extractState reads the previous run's provider state (cursors,
// watermarks) back out of the integration's stored config.
func extractState(config datatypes.JSON) map[string]interface{} {
	var existing map[string]interface{}
	_ = json.Unmarshal(config, &existing)
	state, _ := existing["state"].(map[string]interface{})
	return state
}

// mergeConfig folds a new sync's Details/State into the integration's
// stored config instead of overwriting it wholesale, so provider state
// (like Notion's incremental watermark) survives across runs.
func mergeConfig(oldConfig datatypes.JSON, details, state map[string]interface{}) datatypes.JSON {
	var existing map[string]interface{}
	_ = json.Unmarshal(oldConfig, &existing)
	if existing == nil {
		existing = map[string]interface{}{}
	}

	existingState, _ := existing["state"].(map[string]interface{})
	if existingState == nil {
		existingState = map[string]interface{}{}
	}
	for k, v := range state {
		existingState[k] = v
	}
	existing["state"] = existingState
	existing["details"] = details

	b, err := json.Marshal(existing)
	if err != nil {
		return oldConfig
	}
	return b
}

func (r *integrationSyncRunner) finishRun(ctx context.Context, run *models.SyncRun, status string, records int, warnings []string, syncErr error) {
	if run == nil {
		return
	}
	_ = r.syncRunRepo.Finish(ctx, run.ID, status, records, warnings, syncErr, nil)
}

// ingestContacts upserts each synced contact and its company into the real
// CRM tables (dedupe by domain for companies, by email for contacts) so the
// dashboard is populated the moment a sync completes, not simulated.
func (r *integrationSyncRunner) ingestContacts(ctx context.Context, orgID uuid.UUID, contacts []integration.ContactRecord) error {
	for _, c := range contacts {
		var companyID *uuid.UUID
		if c.CompanyName != "" {
			var company models.Company
			query := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
			if c.CompanyDomain != "" {
				query = query.Where("domain = ?", c.CompanyDomain)
			} else {
				query = query.Where("name = ?", c.CompanyName)
			}
			err := query.First(&company).Error
			if err == gorm.ErrRecordNotFound {
				company = models.Company{
					OrganizationID: orgID,
					Name:           c.CompanyName,
					Status:         "active",
					Source:         strPtr("apollo"),
				}
				if c.CompanyDomain != "" {
					company.Domain = &c.CompanyDomain
				}
				if c.CompanyWebsite != "" {
					company.Website = &c.CompanyWebsite
				}
				if createErr := r.db.WithContext(ctx).Create(&company).Error; createErr != nil {
					return createErr
				}
			} else if err != nil {
				return err
			}
			companyID = &company.ID
		}

		if c.Email == "" && c.FirstName == "" && c.LastName == "" {
			continue
		}

		var existing models.Contact
		err := r.db.WithContext(ctx).Where("organization_id = ? AND email = ?", orgID, c.Email).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			contact := models.Contact{
				OrganizationID: orgID,
				CompanyID:      companyID,
				FirstName:      c.FirstName,
				LastName:       c.LastName,
				Status:         "active",
			}
			if c.Email != "" {
				contact.Email = &c.Email
			}
			if c.Title != "" {
				contact.Title = &c.Title
			}
			if c.LinkedinURL != "" {
				contact.LinkedinURL = &c.LinkedinURL
			}
			if createErr := r.db.WithContext(ctx).Create(&contact).Error; createErr != nil {
				return createErr
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// ingestNotes surfaces synced documents as Activity entries so they show up
// in Recent Activity without needing a dedicated "notes" table.
func (r *integrationSyncRunner) ingestNotes(ctx context.Context, orgID uuid.UUID, provider string, notes []integration.NoteRecord) error {
	for _, n := range notes {
		var description *string
		if n.URL != "" {
			description = &n.URL
		}
		if err := r.db.WithContext(ctx).Create(&models.Activity{
			OrganizationID: orgID,
			EntityType:     "integration_document",
			EntityID:       uuid.New(),
			Type:           provider + "_page_synced",
			Subject:        n.Title,
			Description:    description,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

func (r *integrationSyncRunner) fail(ctx context.Context, rec *models.Integration, syncErr error) error {
	errStr := syncErr.Error()
	r.db.WithContext(ctx).Model(rec).Updates(map[string]interface{}{
		"status":     "error",
		"last_error": errStr,
	})
	return syncErr
}

func (r *integrationSyncRunner) decryptCredentials(raw []byte) (map[string]string, error) {
	var stored struct {
		Enc string `json:"enc"`
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	plain, err := r.cipher.Decrypt(stored.Enc)
	if err != nil {
		return nil, err
	}
	var credentials map[string]string
	if err := json.Unmarshal([]byte(plain), &credentials); err != nil {
		return nil, err
	}
	return credentials, nil
}

func (r *integrationSyncRunner) encryptCredentials(credentials map[string]string) (datatypes.JSON, error) {
	plain, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	enc, err := r.cipher.Encrypt(string(plain))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"enc": enc})
}

func (h *Handlers) HandleIntegrationSync(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	trigger := "manual"
	if payload.Data != nil {
		if tr, ok := payload.Data["trigger"].(string); ok && tr != "" {
			trigger = tr
		}
	}

	h.logger.Info("syncing integration", "org_id", payload.OrgID, "integration_id", payload.EntityID, "provider", payload.Action, "trigger", trigger)

	if err := h.integrationSync.run(ctx, payload.EntityID, trigger); err != nil {
		h.logger.Error("integration sync failed", "integration_id", payload.EntityID, "error", err)
		return err
	}

	h.db.WithContext(ctx).Create(&models.Activity{
		OrganizationID: uuidFromString(payload.OrgID),
		UserID:         uuidPtrFromString(payload.UserID),
		Type:           "integration_synced",
		Subject:        fmt.Sprintf("%s connected and synced", payload.Action),
		EntityType:     "integration",
		EntityID:       uuidFromString(payload.EntityID),
	})
	return nil
}
