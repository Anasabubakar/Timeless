package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Organization struct {
	Base
	Name     string         `gorm:"size:255;not null" json:"name"`
	Slug     string         `gorm:"size:100;uniqueIndex" json:"slug"`
	LogoURL  *string        `gorm:"type:text" json:"logo_url,omitempty"`
	Domain   *string        `gorm:"size:255" json:"domain,omitempty"`
	Plan     string         `gorm:"size:50;not null;default:free" json:"plan"`
	Settings datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	Metadata datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`

	Users []User `gorm:"foreignKey:OrganizationID" json:"users,omitempty"`
	Teams []Team `gorm:"foreignKey:OrganizationID" json:"teams,omitempty"`
}

func (Organization) TableName() string {
	return "organizations"
}

type Team struct {
	Base
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string       `gorm:"size:255;not null" json:"name"`
	Description    *string      `gorm:"type:text" json:"description,omitempty"`
	Organization   Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Members        []TeamMember `gorm:"foreignKey:TeamID" json:"members,omitempty"`
}

func (Team) TableName() string {
	return "teams"
}

type TeamMember struct {
	TeamID uuid.UUID `gorm:"type:uuid;primaryKey" json:"team_id"`
	UserID uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Role   string    `gorm:"size:50;not null;default:member" json:"role"`
	Team   Team      `gorm:"foreignKey:TeamID" json:"-"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`
}

func (TeamMember) TableName() string {
	return "team_members"
}
