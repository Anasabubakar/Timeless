package service

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

// ErrOrgPasswordLocked mirrors ErrAccountLocked for the organization's
// shared join/settings password.
var ErrOrgPasswordLocked = errors.New("too many incorrect organization password attempts, try again later")

// ErrIncorrectOrgPassword is returned when the presented password
// doesn't match — a distinct sentinel (rather than a bare errors.New)
// so callers in different packages can distinguish it with errors.Is
// instead of matching on error text.
var ErrIncorrectOrgPassword = errors.New("incorrect organization password")

// orgPasswordGuard verifies an organization's shared password with
// brute-force lockout, shared by AuthService (signup join flow) and
// OrganizationService (settings changes, ownership transfer) so both
// enforce identical limits against the same counter on the same
// Organization row, rather than each keeping its own separate logic that
// could drift out of sync.
type orgPasswordGuard struct {
	cfg     *config.Config
	orgRepo *repository.OrganizationRepository
	db      *gorm.DB
}

func newOrgPasswordGuard(cfg *config.Config, orgRepo *repository.OrganizationRepository, db *gorm.DB) *orgPasswordGuard {
	return &orgPasswordGuard{cfg: cfg, orgRepo: orgRepo, db: db}
}

func (g *orgPasswordGuard) audit(orgID uuid.UUID, eventType, subject, ip string, metadata map[string]string) {
	middleware.LogSecurityEvent(g.db, orgID, nil, "organization", eventType, subject, ip, metadata)
}

// Verify checks a presented organization password against the stored
// hash, enforcing the same brute-force lockout pattern as user login
// (AuthService.recordFailedLogin) but scoped to the organization as a
// whole: the counter and lock apply to every joiner or settings-change
// attempt against this org, not to one specific user, since the secret
// being guessed is shared.
func (g *orgPasswordGuard) Verify(ctx context.Context, org *models.Organization, password, ip string) error {
	if org.PasswordLockedUntil != nil && time.Now().Before(*org.PasswordLockedUntil) {
		g.audit(org.ID, "org_password_blocked", "organization password attempt while locked", ip, nil)
		return ErrOrgPasswordLocked
	}

	if org.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*org.PasswordHash), []byte(password)) != nil {
		g.recordFailure(ctx, org, ip)
		return ErrIncorrectOrgPassword
	}

	if org.FailedPasswordAttempts > 0 || org.PasswordLockedUntil != nil {
		org.FailedPasswordAttempts = 0
		org.PasswordLockedUntil = nil
		if err := g.orgRepo.Update(ctx, org); err != nil {
			log.Printf("org_password: failed to reset attempt counter for org %s: %v", org.ID, err)
		}
	}
	return nil
}

func (g *orgPasswordGuard) recordFailure(ctx context.Context, org *models.Organization, ip string) {
	org.FailedPasswordAttempts++
	locked := false
	if org.FailedPasswordAttempts >= g.cfg.MaxFailedLogins {
		until := time.Now().Add(g.cfg.LoginLockoutDuration)
		org.PasswordLockedUntil = &until
		locked = true
	}
	g.audit(org.ID, "org_password_failure", "failed organization password attempt", ip, map[string]string{
		"failed_count":            strconv.Itoa(org.FailedPasswordAttempts),
		"organization_now_locked": strconv.FormatBool(locked),
	})
	if err := g.orgRepo.Update(ctx, org); err != nil {
		log.Printf("org_password: failed to persist failed attempt count for org %s: %v", org.ID, err)
	}
}
