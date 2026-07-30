// Package mapping is the reusable sync layer: one internal model, many
// external adapters. Instead of every integration (Notion today; Apollo,
// HubSpot, Airtable, Salesforce tomorrow) hand-rolling its own
// field-by-field translation code, callers describe a record generically
// (SyncableRecord) and a per-org, user-configured FieldMapping, and an
// Adapter turns that into (or back from) the external system's own
// property shape.
package mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
)

// SyncableRecord is the generic, adapter-agnostic representation of one
// internal entity (a Company, Contact, Meeting, ...) for sync purposes.
// Callers build this from their specific Go struct (see
// internal/mapping/extract.go) rather than adapters ever depending on
// concrete model types directly — that's what keeps a new entity type
// or a new external system from needing changes to every existing
// adapter.
type SyncableRecord struct {
	EntityType string
	EntityID   uuid.UUID
	// Fields holds plain Go values (string, float64, bool, time.Time,
	// []string, *string, *float64, *time.Time) keyed by internal field
	// name ("name", "status", "follow_up_date", ...) — never nested
	// structs or provider-specific shapes; that translation is exactly
	// what an Adapter's job is.
	Fields map[string]interface{}
}

// ExternalRecord is the generic representation of what an Adapter reads
// back from (or writes to) the external system: a container ID (Notion
// data source ID, Airtable table ID, ...), an ID for the record itself
// within it, and a raw property payload in that system's own JSON shape
// — Adapter.FromExternal is what turns that raw shape into plain
// SyncableRecord.Fields values.
type ExternalRecord struct {
	ContainerID string
	ExternalID  string
	// RawProperties is the external system's native property JSON (a
	// Notion page's "properties" object, for instance) — opaque outside
	// the adapter that produced/consumes it.
	RawProperties map[string]interface{}
	LastModified  *time.Time
}

// Adapter translates between SyncableRecord and one external system,
// driven entirely by a models.FieldMapping — no entity-type-specific or
// container-specific logic lives in the adapter itself, only
// property-type-specific serialization (how to write a "date" property,
// how to write a "select" property, ...).
type Adapter interface {
	// System names the external system this adapter talks to ("notion").
	System() string

	// ToExternal converts a SyncableRecord into the external system's
	// property shape, applying only the fields the mapping marks for
	// "to_external" or "both" and skipping (not erroring on) any
	// internal field the mapping doesn't cover — an unmapped field is a
	// deliberate configuration choice ("we don't sync this one"), not a
	// failure.
	ToExternal(record SyncableRecord, fm *models.FieldMapping) (map[string]interface{}, error)

	// FromExternal is the inverse: given the external system's raw
	// property payload, produce the internal field values it maps to,
	// applying only "from_external" or "both" fields.
	FromExternal(raw map[string]interface{}, fm *models.FieldMapping) (map[string]interface{}, error)

	// Push creates or updates one record in the external system's
	// container (a Notion database, an Airtable table, ...) and returns
	// its external ID (existingExternalID empty means create, so the
	// caller — which has no other way to learn a freshly-created
	// record's ID — must persist the returned value into the sync
	// ledger).
	Push(ctx context.Context, credentials map[string]string, containerID, existingExternalID string, properties map[string]interface{}, expectedRemoteVersion string) (string, error)

	// Fetch reads one record back from the external system by ID — used
	// by the inbound/conflict-detection path to get the current remote
	// state before deciding how to reconcile a change.
	Fetch(ctx context.Context, credentials map[string]string, containerID, externalID string) (*ExternalRecord, error)

	// Archive removes (soft-deletes where the external system supports
	// it) one record — used when the corresponding internal record is
	// deleted and the mapping should propagate that.
	Archive(ctx context.Context, credentials map[string]string, externalID string) error
}

// ParseFields decodes a FieldMapping's JSON Fields column into the typed
// slice — every caller needs this, so it lives once here rather than
// being re-implemented per adapter.
func ParseFields(fm *models.FieldMapping) ([]models.FieldMappingEntry, error) {
	if fm == nil || len(fm.Fields) == 0 {
		return nil, nil
	}
	var entries []models.FieldMappingEntry
	if err := json.Unmarshal(fm.Fields, &entries); err != nil {
		return nil, fmt.Errorf("parse field mapping: %w", err)
	}
	return entries, nil
}

// entriesForDirection filters a FieldMapping's entries to the ones
// relevant for a given sync direction. "both" always applies; the
// empty string on Direction is treated as "both" too, since an entry a
// user didn't explicitly restrict shouldn't silently stop syncing in
// one direction.
func entriesForDirection(entries []models.FieldMappingEntry, direction string) []models.FieldMappingEntry {
	out := make([]models.FieldMappingEntry, 0, len(entries))
	for _, e := range entries {
		if e.Direction == "" || e.Direction == "both" || e.Direction == direction {
			out = append(out, e)
		}
	}
	return out
}
