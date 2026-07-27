package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type OnboardingService struct {
	repo     *repository.OnboardingRepository
	userRepo *repository.UserRepository
}

func NewOnboardingService(repo *repository.OnboardingRepository, userRepo *repository.UserRepository) *OnboardingService {
	return &OnboardingService{repo: repo, userRepo: userRepo}
}

// GetState returns the user's onboarding progress, creating an empty
// "workspace" step record on first access.
func (s *OnboardingService) GetState(ctx context.Context, orgID, userID uuid.UUID) (*models.OnboardingState, error) {
	state, err := s.repo.GetByUser(ctx, orgID, userID)
	if err == nil {
		return state, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	state = &models.OnboardingState{
		OrganizationID: orgID,
		UserID:         userID,
		Step:           "workspace",
		Payload:        datatypes.JSON([]byte("{}")),
	}
	if err := s.repo.Upsert(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

type SaveStateInput struct {
	Step    string          `json:"step"`
	Payload json.RawMessage `json:"payload"`
}

// SaveState autosaves the payload for a given step, merging it into the
// per-step payload map so earlier steps' data isn't lost.
func (s *OnboardingService) SaveState(ctx context.Context, orgID, userID uuid.UUID, input SaveStateInput) (*models.OnboardingState, error) {
	state, err := s.GetState(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	merged := map[string]json.RawMessage{}
	_ = json.Unmarshal(state.Payload, &merged)
	if input.Step != "" && len(input.Payload) > 0 {
		merged[input.Step] = input.Payload
	}
	mergedBytes, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	state.Payload = datatypes.JSON(mergedBytes)
	if input.Step != "" {
		state.Step = input.Step
	}

	if err := s.repo.Upsert(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// Complete marks the user's onboarding as finished so the dashboard guard
// stops redirecting them back into the wizard.
func (s *OnboardingService) Complete(ctx context.Context, orgID, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.OrganizationID != orgID {
		return nil, gorm.ErrRecordNotFound
	}

	now := time.Now()
	user.OnboardingCompleted = true
	user.OnboardingCompletedAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
