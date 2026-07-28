package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SyncRun is one execution of an integration's sync (initial connect,
// scheduled re-sync, or webhook-triggered incremental sync). The
// Integration Dashboard reads this table directly for history/health.
type SyncRun struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	IntegrationID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"integration_id"`
	Provider       string         `gorm:"size:100;not null;index" json:"provider"`
	Trigger        string         `gorm:"size:50;not null" json:"trigger"`                      // connect | scheduled | webhook | manual | retry
	Status         string         `gorm:"size:50;not null;default:running;index" json:"status"` // running | success | partial | failed
	StartedAt      time.Time      `gorm:"not null" json:"started_at"`
	FinishedAt     *time.Time     `json:"finished_at,omitempty"`
	DurationMS     *int64         `json:"duration_ms,omitempty"`
	RecordsSynced  int            `gorm:"not null;default:0" json:"records_synced"`
	Warnings       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"warnings"`
	Error          *string        `gorm:"type:text" json:"error,omitempty"`
	Attempt        int            `gorm:"not null;default:1" json:"attempt"`
	Details        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"details"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (SyncRun) TableName() string {
	return "sync_runs"
}
