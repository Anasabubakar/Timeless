package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OnboardingState tracks a single user's progress through the post-signup
// onboarding wizard, including autosaved per-step form payloads so progress
// survives refresh/logout before the flow is completed.
type OnboardingState struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Step           string         `gorm:"size:50;not null;default:workspace" json:"step"`
	Payload        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	User         User         `gorm:"foreignKey:UserID" json:"-"`
}

func (OnboardingState) TableName() string {
	return "onboarding_states"
}
