package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Project struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Status         string         `gorm:"size:50;not null;default:active" json:"status"`
	StartDate      *string        `gorm:"type:date" json:"start_date,omitempty"`
	EndDate        *string        `gorm:"type:date" json:"end_date,omitempty"`
	Budget         *float64       `gorm:"type:numeric(15,2)" json:"budget,omitempty"`
	Currency       string         `gorm:"size:3;not null;default:USD" json:"currency"`
	Settings       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Campaigns    []Campaign   `gorm:"foreignKey:ProjectID" json:"campaigns,omitempty"`
}

func (Project) TableName() string {
	return "projects"
}

type Campaign struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	ProjectID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Status         string         `gorm:"size:50;not null;default:draft" json:"status"`
	GoalAmount     *float64       `gorm:"type:numeric(15,2)" json:"goal_amount,omitempty"`
	RaisedAmount   float64        `gorm:"type:numeric(15,2);not null;default:0" json:"raised_amount"`
	Currency       string         `gorm:"size:3;not null;default:USD" json:"currency"`
	StartDate      *string        `gorm:"type:date" json:"start_date,omitempty"`
	EndDate        *string        `gorm:"type:date" json:"end_date,omitempty"`
	PipelineStages datatypes.JSON `gorm:"type:jsonb;not null" json:"pipeline_stages"`
	Settings       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Project      Project      `gorm:"foreignKey:ProjectID" json:"-"`
	Sponsors     []Sponsor    `gorm:"foreignKey:CampaignID" json:"sponsors,omitempty"`
}

func (Campaign) TableName() string {
	return "campaigns"
}

var DefaultPipelineStages = []string{
	"prospect",
	"contacted",
	"qualified",
	"proposal",
	"negotiation",
	"closed_won",
	"closed_lost",
}
