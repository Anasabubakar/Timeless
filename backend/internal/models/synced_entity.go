package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SyncedEntity is the per-record bidirectional-sync ledger: one row per
// (internal entity, external system) pair. Where SyncRun records "an
// integration ran, here's what happened in aggregate", SyncedEntity
// tracks the specific state of one record so conflict resolution has
// something to compare against — what did we last see on each side, and
// who changed it most recently.
type SyncedEntity struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index:idx_synced_entity_org" json:"organization_id"`
	IntegrationID  uuid.UUID `gorm:"type:uuid;not null;index" json:"integration_id"`

	// EntityType/EntityID identify the internal record ("company",
	// "contact", "meeting", "task", ...) — deliberately a string type
	// name + UUID pair rather than a foreign key to one specific table,
	// since this ledger covers every syncable entity type uniformly.
	EntityType string    `gorm:"size:50;not null;index:idx_synced_entity_internal" json:"entity_type"`
	EntityID   uuid.UUID `gorm:"type:uuid;not null;index:idx_synced_entity_internal" json:"entity_id"`

	// ExternalSystem + ExternalID identify the far side ("notion",
	// "apollo", ...) and the record within it (a Notion page ID, etc).
	ExternalSystem string `gorm:"size:50;not null" json:"external_system"`
	ExternalID     string `gorm:"size:255;not null;index:idx_synced_entity_external" json:"external_id"`

	// SyncState: "synced" (both sides match as of LastSyncedAt),
	// "pending" (a local or remote change hasn't been pushed yet),
	// "conflict" (both sides changed since the last sync and need
	// resolution — see ConflictState), "error" (last sync attempt
	// failed — see LastError).
	SyncState string `gorm:"size:20;not null;default:pending;index" json:"sync_state"`

	// Version increments on every successful sync in either direction —
	// a simple logical clock independent of wall-clock time, which
	// timestamps alone can't be trusted for across systems with
	// potentially skewed clocks.
	Version int `gorm:"not null;default:0" json:"version"`

	// Source records which side produced the most recent change this
	// ledger row has observed: "timeless" or the external system's name.
	// Used to break ties when both LastModifiedLocal and
	// LastModifiedRemote advance without an intervening sync.
	Source string `gorm:"size:50;not null;default:timeless" json:"source"`

	LastModifiedLocal  *time.Time `json:"last_modified_local,omitempty"`
	LastModifiedRemote *time.Time `json:"last_modified_remote,omitempty"`
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`

	// ConflictState is empty when SyncState != "conflict"; otherwise one
	// of "local_newer", "remote_newer", "both_changed" — set by the
	// conflict detector, cleared once ConflictResolution records how it
	// was resolved.
	ConflictState      string         `gorm:"size:20" json:"conflict_state,omitempty"`
	ConflictResolution *string        `gorm:"size:20" json:"conflict_resolution,omitempty"` // "kept_local" | "kept_remote" | "merged"
	ConflictDetails    datatypes.JSON `gorm:"type:jsonb" json:"conflict_details,omitempty"`

	LastError *string `gorm:"type:text" json:"last_error,omitempty"`

	// FieldMappingID references the mapping config used for this
	// entity's property translation — nil until the mapping engine
	// (next phase) assigns one.
	FieldMappingID *uuid.UUID `gorm:"type:uuid" json:"field_mapping_id,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Integration  Integration  `gorm:"foreignKey:IntegrationID" json:"-"`
}

func (SyncedEntity) TableName() string {
	return "synced_entities"
}

// SyncHistory is an append-only log of every sync action taken on a
// SyncedEntity — pushed, pulled, conflict detected, conflict resolved.
// Separate from SyncedEntity itself (which holds only current state) so
// "what happened to this record over time" is answerable without
// mutating the ledger row.
type SyncHistory struct {
	Base
	SyncedEntityID uuid.UUID `gorm:"type:uuid;not null;index" json:"synced_entity_id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`

	// Action: "pushed_to_remote" | "pulled_from_remote" | "conflict_detected"
	// | "conflict_resolved" | "sync_failed".
	Action  string         `gorm:"size:50;not null" json:"action"`
	Source  string         `gorm:"size:50;not null" json:"source"`
	Details datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
	Error   *string        `gorm:"type:text" json:"error,omitempty"`

	SyncedEntity SyncedEntity `gorm:"foreignKey:SyncedEntityID" json:"-"`
}

func (SyncHistory) TableName() string {
	return "sync_history"
}
