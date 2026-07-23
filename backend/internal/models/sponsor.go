package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Sponsor struct {
	Base
	OrganizationID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	CampaignID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"campaign_id"`
	CompanyID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"company_id"`
	Stage            string         `gorm:"size:50;not null;default:prospect" json:"stage"`
	StageEnteredAt   time.Time      `gorm:"not null;default:now()" json:"stage_entered_at"`
	DealValue        *float64       `gorm:"type:numeric(15,2)" json:"deal_value,omitempty"`
	Probability      *int           `json:"probability,omitempty"`
	Tier             *string        `gorm:"size:50" json:"tier,omitempty"`
	AssignedTo       *uuid.UUID     `gorm:"type:uuid" json:"assigned_to,omitempty"`
	PrimaryContactID *uuid.UUID     `gorm:"type:uuid" json:"primary_contact_id,omitempty"`
	ExpectedClose    *string        `gorm:"type:date" json:"expected_close,omitempty"`
	ActualClose      *string        `gorm:"type:date" json:"actual_close,omitempty"`
	LostReason       *string        `gorm:"type:text" json:"lost_reason,omitempty"`
	Notes            *string        `gorm:"type:text" json:"notes,omitempty"`
	CustomFields     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"custom_fields"`
	Position         int            `gorm:"not null;default:0" json:"position"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Campaign     Campaign     `gorm:"foreignKey:CampaignID" json:"campaign,omitempty"`
	Company      Company      `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
	AssignedUser *User        `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
	Proposals    []Proposal   `gorm:"foreignKey:SponsorID" json:"proposals,omitempty"`
}

func (Sponsor) TableName() string {
	return "sponsors"
}

type Proposal struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	SponsorID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"sponsor_id"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Content        *string        `gorm:"type:text" json:"content,omitempty"`
	Status         string         `gorm:"size:50;not null;default:draft" json:"status"`
	Version        int            `gorm:"not null;default:1" json:"version"`
	Amount         *float64       `gorm:"type:numeric(15,2)" json:"amount,omitempty"`
	ValidUntil     *string        `gorm:"type:date" json:"valid_until,omitempty"`
	SentAt         *time.Time     `json:"sent_at,omitempty"`
	ViewedAt       *time.Time     `json:"viewed_at,omitempty"`
	RespondedAt    *time.Time     `json:"responded_at,omitempty"`
	Attachments    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"attachments"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Sponsor      Sponsor      `gorm:"foreignKey:SponsorID" json:"-"`
}

func (Proposal) TableName() string {
	return "proposals"
}
