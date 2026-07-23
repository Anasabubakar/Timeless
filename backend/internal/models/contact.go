package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Contact struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	CompanyID      *uuid.UUID     `gorm:"type:uuid;index" json:"company_id,omitempty"`
	FirstName      string         `gorm:"size:100;not null" json:"first_name"`
	LastName       string         `gorm:"size:100;not null" json:"last_name"`
	Email          *string        `gorm:"size:255" json:"email,omitempty"`
	Phone          *string        `gorm:"size:50" json:"phone,omitempty"`
	Title          *string        `gorm:"size:255" json:"title,omitempty"`
	Department     *string        `gorm:"size:100" json:"department,omitempty"`
	LinkedinURL    *string        `gorm:"type:text" json:"linkedin_url,omitempty"`
	AvatarURL      *string        `gorm:"type:text" json:"avatar_url,omitempty"`
	Notes          *string        `gorm:"type:text" json:"notes,omitempty"`
	Tags           pq.StringArray `gorm:"type:text[];not null;default:'{}'" json:"tags"`
	CustomFields   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"custom_fields"`
	LastContactedAt *time.Time    `json:"last_contacted_at,omitempty"`
	Status         string         `gorm:"size:50;not null;default:active" json:"status"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Company      *Company     `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
}

func (Contact) TableName() string {
	return "contacts"
}
