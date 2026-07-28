package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Integration struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Provider       string         `gorm:"size:100;not null" json:"provider"`
	Type           string         `gorm:"size:50;not null" json:"type"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Status         string         `gorm:"size:50;not null;default:inactive" json:"status"`
	Credentials    datatypes.JSON `gorm:"type:jsonb" json:"-"`
	Config         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	LastSyncAt     *time.Time     `json:"last_sync_at,omitempty"`
	LastError      *string        `gorm:"type:text" json:"last_error,omitempty"`
	WebhookURL     *string        `gorm:"type:text" json:"webhook_url,omitempty"`
	WebhookSecret  *string        `gorm:"type:text" json:"-"`
	InstalledBy    *uuid.UUID     `gorm:"type:uuid" json:"installed_by,omitempty"`
	// ExternalAccountID is the provider's own account/workspace id (e.g.
	// Notion's workspace_id). It's not a secret, so it's stored in the
	// clear (unlike Credentials) specifically so inbound webhook events —
	// which identify the workspace but carry no org context of ours — can
	// be routed back to the right Integration row.
	ExternalAccountID *string `gorm:"size:255;index" json:"external_account_id,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (Integration) TableName() string {
	return "integrations"
}

type Webhook struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	IntegrationID  *uuid.UUID     `gorm:"type:uuid" json:"integration_id,omitempty"`
	URL            string         `gorm:"type:text;not null" json:"url"`
	Events         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"events"`
	Secret         string         `gorm:"size:255;not null" json:"-"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	LastTriggered  *time.Time     `json:"last_triggered,omitempty"`
	FailureCount   int            `gorm:"not null;default:0" json:"failure_count"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (Webhook) TableName() string {
	return "webhooks"
}

type Automation struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	TriggerType    string         `gorm:"size:50;not null" json:"trigger_type"`
	TriggerConfig  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"trigger_config"`
	Actions        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"actions"`
	Conditions     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"conditions"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	RunCount       int            `gorm:"not null;default:0" json:"run_count"`
	LastRunAt      *time.Time     `json:"last_run_at,omitempty"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (Automation) TableName() string {
	return "automations"
}
