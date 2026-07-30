package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type InvitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Create(ctx context.Context, inv *models.Invitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

// FindPendingByTokenHash looks up an invitation still awaiting acceptance
// — not yet accepted, not revoked, not expired — the only state an
// accept attempt should ever succeed against.
func (r *InvitationRepository) FindPendingByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) FindByID(ctx context.Context, id, orgID uuid.UUID) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListPending returns every not-yet-accepted, not-yet-revoked,
// not-yet-expired invitation for an organization, newest first.
func (r *InvitationRepository) ListPending(ctx context.Context, orgID uuid.UUID) ([]models.Invitation, error) {
	var invs []models.Invitation
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", orgID, time.Now()).
		Order("created_at DESC").
		Find(&invs).Error
	return invs, err
}

// FindActiveByEmail finds a still-pending invitation for an email within
// an org, if any — used to avoid double-inviting the same address.
func (r *InvitationRepository) FindActiveByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND email = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", orgID, email, time.Now()).
		First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Invitation{}).Where("id = ?", id).
		Update("accepted_at", time.Now()).Error
}

func (r *InvitationRepository) Revoke(ctx context.Context, id, orgID uuid.UUID) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Invitation{}).
		Where("id = ? AND organization_id = ? AND accepted_at IS NULL AND revoked_at IS NULL", id, orgID).
		Update("revoked_at", time.Now())
	return result.RowsAffected, result.Error
}
