package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type User struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Email          string         `gorm:"size:255;not null" json:"email"`
	PasswordHash   *string        `gorm:"size:255" json:"-"`
	FirstName      string         `gorm:"size:100;not null" json:"first_name"`
	LastName       string         `gorm:"size:100;not null" json:"last_name"`
	AvatarURL      *string        `gorm:"type:text" json:"avatar_url,omitempty"`
	Phone          *string        `gorm:"size:50" json:"phone,omitempty"`
	JobTitle       *string        `gorm:"size:255" json:"job_title,omitempty"`
	Status         string         `gorm:"size:20;not null;default:active" json:"status"`
	LastLoginAt    *time.Time     `json:"last_login_at,omitempty"`
	Preferences    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"preferences"`

	Organization  Organization   `gorm:"foreignKey:OrganizationID" json:"-"`
	Roles         []Role         `gorm:"many2many:user_roles" json:"roles,omitempty"`
	OAuthAccounts []OAuthAccount `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

type OAuthAccount struct {
	Base
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider     string     `gorm:"size:50;not null" json:"provider"`
	ProviderID   string     `gorm:"size:255;not null" json:"provider_id"`
	AccessToken  *string    `gorm:"type:text" json:"-"`
	RefreshToken *string    `gorm:"type:text" json:"-"`
	TokenExpires *time.Time `json:"-"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
}

func (OAuthAccount) TableName() string {
	return "oauth_accounts"
}

type Role struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Permissions    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"permissions"`
	IsSystem       bool           `gorm:"not null;default:false" json:"is_system"`
	Organization   Organization   `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string    `gorm:"size:500;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
