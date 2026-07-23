package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Activity struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	EntityType     string         `gorm:"size:50;not null" json:"entity_type"`
	EntityID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"entity_id"`
	Type           string         `gorm:"column:action;size:50;not null" json:"type"`
	Subject        string         `gorm:"size:500;not null;default:''" json:"subject"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Metadata       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	IPAddress      *string        `gorm:"size:45" json:"ip_address,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	User         *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Activity) TableName() string {
	return "activities"
}

type Communication struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	SponsorID      *uuid.UUID     `gorm:"type:uuid;index" json:"sponsor_id,omitempty"`
	ContactID      *uuid.UUID     `gorm:"type:uuid" json:"contact_id,omitempty"`
	Type           string         `gorm:"size:50;not null" json:"type"`
	Direction      string         `gorm:"size:10;not null;default:outbound" json:"direction"`
	Subject        *string        `gorm:"size:500" json:"subject,omitempty"`
	Body           *string        `gorm:"type:text" json:"body,omitempty"`
	Status         string         `gorm:"size:50;not null;default:draft" json:"status"`
	SentAt         *time.Time     `json:"sent_at,omitempty"`
	OpenedAt       *time.Time     `json:"opened_at,omitempty"`
	ClickedAt      *time.Time     `json:"clicked_at,omitempty"`
	RepliedAt      *time.Time     `json:"replied_at,omitempty"`
	BouncedAt      *time.Time     `json:"bounced_at,omitempty"`
	Channel        *string        `gorm:"size:50" json:"channel,omitempty"`
	ExternalID     *string        `gorm:"size:255" json:"external_id,omitempty"`
	TemplateID     *uuid.UUID     `gorm:"type:uuid" json:"template_id,omitempty"`
	Metadata       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	SentBy         *uuid.UUID     `gorm:"type:uuid" json:"sent_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (Communication) TableName() string {
	return "communications"
}

type EmailTemplate struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Subject        string         `gorm:"size:500;not null" json:"subject"`
	Body           string         `gorm:"type:text;not null" json:"body"`
	Category       *string        `gorm:"size:100" json:"category,omitempty"`
	Variables      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"variables"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}

type Task struct {
	Base
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	Title          string     `gorm:"size:500;not null" json:"title"`
	Description    *string    `gorm:"type:text" json:"description,omitempty"`
	Status         string     `gorm:"size:50;not null;default:pending" json:"status"`
	Priority       string     `gorm:"size:20;not null;default:medium" json:"priority"`
	DueDate        *string    `gorm:"type:date" json:"due_date,omitempty"`
	AssignedTo     *uuid.UUID `gorm:"type:uuid" json:"assigned_to,omitempty"`
	EntityType     *string    `gorm:"size:50" json:"entity_type,omitempty"`
	EntityID       *uuid.UUID `gorm:"type:uuid" json:"entity_id,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Assignee     *User        `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}
