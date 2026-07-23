package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Company struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Domain         *string        `gorm:"size:255" json:"domain,omitempty"`
	Website        *string        `gorm:"type:text" json:"website,omitempty"`
	LogoURL        *string        `gorm:"type:text" json:"logo_url,omitempty"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	IndustryID     *uuid.UUID     `gorm:"type:uuid" json:"industry_id,omitempty"`
	EmployeeCount  *string        `gorm:"size:50" json:"employee_count,omitempty"`
	AnnualRevenue  *string        `gorm:"size:100" json:"annual_revenue,omitempty"`
	Headquarters   *string        `gorm:"size:255" json:"headquarters,omitempty"`
	FoundedYear    *int           `json:"founded_year,omitempty"`
	LinkedinURL    *string        `gorm:"type:text" json:"linkedin_url,omitempty"`
	TwitterURL     *string        `gorm:"type:text" json:"twitter_url,omitempty"`
	Phone          *string        `gorm:"size:50" json:"phone,omitempty"`
	Address        datatypes.JSON `gorm:"type:jsonb" json:"address,omitempty"`
	Tags           pq.StringArray `gorm:"type:text[];not null;default:'{}'" json:"tags"`
	EnrichmentData datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"enrichment_data"`
	Score          *int           `json:"score,omitempty"`
	Status         string         `gorm:"size:50;not null;default:active" json:"status"`
	Source         *string        `gorm:"size:100" json:"source,omitempty"`

	Organization   Organization    `gorm:"foreignKey:OrganizationID" json:"-"`
	Industry       *Industry       `gorm:"foreignKey:IndustryID" json:"industry,omitempty"`
	DecisionMakers []DecisionMaker `gorm:"foreignKey:CompanyID" json:"decision_makers,omitempty"`
	PainPoints     []PainPoint     `gorm:"foreignKey:CompanyID" json:"pain_points,omitempty"`
}

func (Company) TableName() string {
	return "companies"
}

type Industry struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name     string     `gorm:"size:255;not null;uniqueIndex" json:"name"`
	Slug     string     `gorm:"size:100;not null;uniqueIndex" json:"slug"`
	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *Industry  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
}

func (Industry) TableName() string {
	return "industries"
}

type PainPoint struct {
	Base
	CompanyID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"company_id"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Severity    *string        `gorm:"size:20" json:"severity,omitempty"`
	Source      *string        `gorm:"size:50" json:"source,omitempty"`
	Evidence    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"evidence"`
	Company     Company        `gorm:"foreignKey:CompanyID" json:"-"`
}

func (PainPoint) TableName() string {
	return "pain_points"
}

type DecisionMaker struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	CompanyID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"company_id"`
	FirstName      string         `gorm:"size:100;not null" json:"first_name"`
	LastName       string         `gorm:"size:100;not null" json:"last_name"`
	Title          *string        `gorm:"size:255" json:"title,omitempty"`
	Department     *string        `gorm:"size:255" json:"department,omitempty"`
	Email          *string        `gorm:"size:255" json:"email,omitempty"`
	Phone          *string        `gorm:"size:50" json:"phone,omitempty"`
	LinkedinURL    *string        `gorm:"type:text" json:"linkedin_url,omitempty"`
	TwitterURL     *string        `gorm:"type:text" json:"twitter_url,omitempty"`
	InfluenceLevel *string        `gorm:"size:20" json:"influence_level,omitempty"`
	Bio            *string        `gorm:"type:text" json:"bio,omitempty"`
	ProfileData    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"profile_data"`
	AvatarURL      *string        `gorm:"type:text" json:"avatar_url,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Company      Company      `gorm:"foreignKey:CompanyID" json:"-"`
}

func (DecisionMaker) TableName() string {
	return "decision_makers"
}
