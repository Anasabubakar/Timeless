package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Organization struct {
	Base
	Name    string  `gorm:"size:255;not null" json:"name"`
	Slug    string  `gorm:"size:100;uniqueIndex" json:"slug"`
	LogoURL *string `gorm:"type:text" json:"logo_url,omitempty"`
	Domain  *string `gorm:"size:255" json:"domain,omitempty"`
	Plan    string  `gorm:"size:50;not null;default:free" json:"plan"`

	// PasswordHash gates joining an existing organization at signup and
	// re-authorizes identity-changing settings updates (rename, slug,
	// password rotation, ownership transfer) — see
	// OrganizationService.VerifyPassword. Never nil in practice (Register
	// always sets it), pointer only so the zero value can't be mistaken
	// for a real bcrypt hash.
	PasswordHash *string `gorm:"size:255" json:"-"`
	// Brute-force protection on the org password, mirroring
	// User.FailedLoginCount/LockedUntil — attempts are scoped to the
	// organization (not per-user), since the shared org password is a
	// single secret every joiner and every settings change checks.
	FailedPasswordAttempts int        `gorm:"not null;default:0" json:"-"`
	PasswordLockedUntil    *time.Time `json:"-"`

	Settings datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	Metadata datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`

	Users       []User       `gorm:"foreignKey:OrganizationID" json:"users,omitempty"`
	Teams       []Team       `gorm:"foreignKey:OrganizationID" json:"teams,omitempty"`
	Invitations []Invitation `gorm:"foreignKey:OrganizationID" json:"invitations,omitempty"`
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

// Invitation is a pending offer to join an organization at a specific
// role, emailed as a link containing the raw token. Only TokenHash
// (sha256) is persisted — same treatment as EmailVerificationToken/
// PasswordResetToken — so a database read alone can't be used to accept
// someone else's invitation. The invited person has no User row at all
// until they accept: unlike the old flow this replaces (which created a
// User immediately with no password, so the person could never actually
// log in), nothing exists for them to be locked out of before they've
// agreed to anything.
type Invitation struct {
	Base
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	Email          string     `gorm:"size:255;not null;index" json:"email"`
	Role           string     `gorm:"size:100;not null" json:"role"`
	TokenHash      string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	InvitedByID    uuid.UUID  `gorm:"type:uuid;not null" json:"invited_by_id"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	InvitedBy    User         `gorm:"foreignKey:InvitedByID" json:"-"`
}

func (Invitation) TableName() string {
	return "invitations"
}
