package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type OnboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

func (r *OnboardingRepository) GetByUser(ctx context.Context, orgID, userID uuid.UUID) (*models.OnboardingState, error) {
	var state models.OnboardingState
	err := r.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// Upsert creates the state row if it doesn't exist yet, otherwise updates
// the step and payload of the existing row for this user.
func (r *OnboardingRepository) Upsert(ctx context.Context, state *models.OnboardingState) error {
	var existing models.OnboardingState
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", state.OrganizationID, state.UserID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(state).Error
	}
	if err != nil {
		return err
	}

	existing.Step = state.Step
	existing.Payload = state.Payload
	if saveErr := r.db.WithContext(ctx).Save(&existing).Error; saveErr != nil {
		return saveErr
	}
	*state = existing
	return nil
}
