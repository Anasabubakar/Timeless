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

	// Email verification. New accounts start unverified; Status stays
	// "active" so existing login/session logic is unaffected, but
	// verification-gated actions can check EmailVerified explicitly.
	EmailVerified   bool       `gorm:"not null;default:false" json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`

	// Brute-force protection. FailedLoginCount resets to 0 on any
	// successful login; LockedUntil is set once the count crosses the
	// configured threshold and cleared automatically once it elapses.
	FailedLoginCount int        `gorm:"not null;default:0" json:"-"`
	LockedUntil      *time.Time `json:"-"`

	// TOTP-based MFA. MFASecretEncrypted is encrypted at rest with the same
	// CredentialCipher used for OAuth tokens. BackupCodesHashed stores
	// bcrypt hashes, never the plaintext codes (those are shown once at
	// enrollment time only).
	MFAEnabled         bool           `gorm:"not null;default:false" json:"mfa_enabled"`
	MFASecretEncrypted *string        `gorm:"type:text" json:"-"`
	MFABackupCodesHash datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"-"`
	MFAEnrolledAt      *time.Time     `json:"mfa_enrolled_at,omitempty"`

	OnboardingCompleted   bool       `gorm:"not null;default:false" json:"onboarding_completed"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at,omitempty"`

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

// RefreshToken doubles as the durable session record for a device: one row
// per issued refresh token, so "logout everywhere" and "list my sessions"
// both have something to query/revoke instead of relying solely on the
// Redis blacklist (which only ever answers "is this one token revoked").
type RefreshToken struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash   string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	DeviceLabel *string    `gorm:"size:255" json:"device_label,omitempty"`
	IP          *string    `gorm:"size:64" json:"ip,omitempty"`
	UserAgent   *string    `gorm:"size:500" json:"user_agent,omitempty"`
	RememberMe  bool       `gorm:"not null;default:false" json:"remember_me"`
	LastUsedAt  time.Time  `gorm:"not null" json:"last_used_at"`
	ExpiresAt   time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	User        User       `gorm:"foreignKey:UserID" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// EmailVerificationToken is a single-use, expiring token emailed to a user
// to prove ownership of their address. Only TokenHash (sha256) is stored so
// a DB read alone can't be used to verify someone else's account.
type EmailVerificationToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}

func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}

// PasswordResetToken is a single-use, expiring token emailed to a user who
// requested a password reset. Same hash-at-rest treatment as
// EmailVerificationToken, plus IssuedIP for audit/anomaly review.
type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	IssuedIP  *string    `gorm:"size:64" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
