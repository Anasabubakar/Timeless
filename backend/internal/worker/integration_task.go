package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/security"
)

// integrationSyncRunner decrypts a connected integration's stored
// credentials and performs a real sync call against the provider.
type integrationSyncRunner struct {
	db      *gorm.DB
	cipher  *security.CredentialCipher
	clients map[string]integration.Client
}

func newIntegrationSyncRunner(db *gorm.DB, cipher *security.CredentialCipher) *integrationSyncRunner {
	return &integrationSyncRunner{db: db, cipher: cipher, clients: integration.Registry()}
}

func (r *integrationSyncRunner) run(ctx context.Context, integrationID string) error {
	var rec models.Integration
	if err := r.db.WithContext(ctx).Where("id = ?", integrationID).First(&rec).Error; err != nil {
		return fmt.Errorf("find integration: %w", err)
	}

	client, ok := r.clients[rec.Provider]
	if !ok {
		return fmt.Errorf("unsupported provider %q", rec.Provider)
	}

	credentials, err := r.decryptCredentials(rec.Credentials)
	if err != nil {
		return r.fail(ctx, &rec, fmt.Errorf("decrypt credentials: %w", err))
	}

	result, err := client.Sync(ctx, credentials)
	if err != nil {
		return r.fail(ctx, &rec, err)
	}

	if err := r.ingestContacts(ctx, rec.OrganizationID, result.Contacts); err != nil {
		return r.fail(ctx, &rec, fmt.Errorf("ingest contacts: %w", err))
	}
	if err := r.ingestNotes(ctx, rec.OrganizationID, rec.Provider, result.Notes); err != nil {
		return r.fail(ctx, &rec, fmt.Errorf("ingest notes: %w", err))
	}

	details, _ := json.Marshal(result.Details)
	now := time.Now()
	return r.db.WithContext(ctx).Model(&rec).Updates(map[string]interface{}{
		"status":       "active",
		"last_sync_at": now,
		"last_error":   nil,
		"config":       details,
	}).Error
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

func (h *Handlers) HandleIntegrationSync(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	h.logger.Info("syncing integration", "org_id", payload.OrgID, "integration_id", payload.EntityID, "provider", payload.Action)

	if err := h.integrationSync.run(ctx, payload.EntityID); err != nil {
		h.logger.Error("integration sync failed", "integration_id", payload.EntityID, "error", err)
		return nil
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
