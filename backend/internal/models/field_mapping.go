package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// FieldMapping is a user-configured translation between one internal
// entity type ("company", "contact", "meeting", ...) and one external
// database/object in one integration — never a hardcoded database ID:
// OrgID+IntegrationID+EntityType+ExternalContainerID together identify
// exactly one mapping, and Fields (a JSON array of {internal_field,
// external_field, external_type, direction}) is entirely user-defined
// through the mapping engine's config API, so this works against
// whatever Notion database (or, later, HubSpot/Airtable/Salesforce
// object) the user actually has — not one this app assumes exists.
type FieldMapping struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index:idx_field_mapping_lookup" json:"organization_id"`
	IntegrationID  uuid.UUID `gorm:"type:uuid;not null;index:idx_field_mapping_lookup" json:"integration_id"`
	EntityType     string    `gorm:"size:50;not null;index:idx_field_mapping_lookup" json:"entity_type"`

	// ExternalContainerID is the external system's own identifier for
	// "where these records live" — a Notion database/data source ID
	// today; an Airtable base+table ID or a Salesforce object API name
	// tomorrow. User-discovered and user-selected, never assumed.
	ExternalContainerID string `gorm:"size:255;not null" json:"external_container_id"`

	// Fields is a []FieldMappingEntry, JSON-encoded — see that type.
	Fields datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"fields"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Integration  Integration  `gorm:"foreignKey:IntegrationID" json:"-"`
}

func (FieldMapping) TableName() string {
	return "field_mappings"
}

// FieldMappingEntry is one field-level rule within a FieldMapping.Fields
// array.
type FieldMappingEntry struct {
	InternalField string `json:"internal_field"` // e.g. "name", "status", "follow_up_date"
	ExternalField string `json:"external_field"` // e.g. Notion property name
	// ExternalType names the external property's type in that system's
	// own vocabulary — for Notion: title, rich_text, select,
	// multi_select, date, number, checkbox, people, relation, status,
	// url, email, phone_number. Drives which serializer/deserializer
	// the adapter uses for this field.
	ExternalType string `json:"external_type"`
	// Direction: "both" (default), "to_external" (Timeless -> external
	// only, e.g. a computed/internal-only field), or "from_external"
	// (external -> Timeless only, e.g. a field only the external system
	// should own).
	Direction string `json:"direction"`
}
