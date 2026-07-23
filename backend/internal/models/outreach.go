package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OutreachSequence struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Status         string         `gorm:"size:50;not null;default:active" json:"status"`
	TriggerType    *string        `gorm:"size:100" json:"trigger_type,omitempty"`
	TriggerConfig  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"trigger_config"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization     `gorm:"foreignKey:OrganizationID" json:"-"`
	Steps        []SequenceStep   `gorm:"foreignKey:SequenceID" json:"steps,omitempty"`
	Enrollments  []Enrollment     `gorm:"foreignKey:SequenceID" json:"enrollments,omitempty"`
}

func (OutreachSequence) TableName() string {
	return "outreach_sequences"
}

type SequenceStep struct {
	Base
	SequenceID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"sequence_id"`
	Position       int            `gorm:"not null;default:0" json:"position"`
	Type           string         `gorm:"size:50;not null;default:email" json:"type"`
	Subject        *string        `gorm:"size:500" json:"subject,omitempty"`
	Body           *string        `gorm:"type:text" json:"body,omitempty"`
	TemplateID     *uuid.UUID     `gorm:"type:uuid" json:"template_id,omitempty"`
	DelayDays      int            `gorm:"not null;default:1" json:"delay_days"`
	DelayHours     int            `gorm:"not null;default:0" json:"delay_hours"`
	Conditions     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"conditions"`

	Sequence OutreachSequence `gorm:"foreignKey:SequenceID" json:"-"`
}

func (SequenceStep) TableName() string {
	return "sequence_steps"
}

type Enrollment struct {
	Base
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	SequenceID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"sequence_id"`
	ContactID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	SponsorID      *uuid.UUID `gorm:"type:uuid" json:"sponsor_id,omitempty"`
	CurrentStep    int        `gorm:"not null;default:0" json:"current_step"`
	Status         string     `gorm:"size:50;not null;default:active" json:"status"`
	PausedAt       *time.Time `json:"paused_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at,omitempty"`

	Organization Organization     `gorm:"foreignKey:OrganizationID" json:"-"`
	Sequence     OutreachSequence `gorm:"foreignKey:SequenceID" json:"sequence,omitempty"`
	Contact      Contact          `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
}

func (Enrollment) TableName() string {
	return "enrollments"
}
