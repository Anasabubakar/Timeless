package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

// SessionRepository persists refresh-token-backed sessions: one row per
// device/login, so "list my sessions" and "log out everywhere" have
// something durable to query instead of relying solely on a Redis
// blacklist keyed by the token that's being revoked.
type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var s models.RefreshToken
	err := r.db.WithContext(ctx).First(&s, "token_hash = ?", hash).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListActiveByUser returns sessions that are neither revoked nor expired,
// most-recently-used first, for the "your devices" settings view.
func (r *SessionRepository) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]models.RefreshToken, error) {
	var sessions []models.RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("last_used_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *SessionRepository) Touch(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllForUser powers "log out everywhere" (e.g. after a password
// reset or a user-initiated "sign out of all devices").
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// EmailVerificationRepository stores single-use email-verification tokens.
type EmailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(db *gorm.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, t *models.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *EmailVerificationRepository) FindByTokenHash(ctx context.Context, hash string) (*models.EmailVerificationToken, error) {
	var t models.EmailVerificationToken
	err := r.db.WithContext(ctx).First(&t, "token_hash = ?", hash).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *EmailVerificationRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.EmailVerificationToken{}).
		Where("id = ?", id).
		Update("used_at", time.Now()).Error
}

// InvalidateAllForUser is called whenever a new verification token is
// issued or the address is verified, so stale tokens can't be replayed.
func (r *EmailVerificationRepository) InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Delete(&models.EmailVerificationToken{}).Error
}

// PasswordResetRepository stores single-use password-reset tokens.
type PasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, t *models.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *PasswordResetRepository) FindByTokenHash(ctx context.Context, hash string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := r.db.WithContext(ctx).First(&t, "token_hash = ?", hash).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", time.Now()).Error
}

func (r *PasswordResetRepository) InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Delete(&models.PasswordResetToken{}).Error
}
