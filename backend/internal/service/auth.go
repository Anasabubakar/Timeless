package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/email"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
)

// maxSlugAttempts bounds the unique-org-slug retry loop in Register — a
// generous ceiling for what should almost always resolve on the first or
// second try; existing purely as a backstop against an unbounded loop if
// something is systematically wrong (e.g. the slug column itself broken).
const maxSlugAttempts = 20

type AuthService struct {
	userRepo        *repository.UserRepository
	orgRepo         *repository.OrganizationRepository
	sessionRepo     *repository.SessionRepository
	emailVerifyRepo *repository.EmailVerificationRepository
	resetRepo       *repository.PasswordResetRepository
	roleRepo        *repository.RoleRepository
	cfg             *config.Config
	rdb             *redis.Client
	mailer          *email.Sender
	cipher          *security.CredentialCipher
	keyring         *security.JWTKeyring
	// db is used only for security-event audit logging
	// (middleware.LogSecurityEvent) — every other read/write goes
	// through the repositories above.
	db *gorm.DB
}

func NewAuthService(
	userRepo *repository.UserRepository,
	orgRepo *repository.OrganizationRepository,
	sessionRepo *repository.SessionRepository,
	emailVerifyRepo *repository.EmailVerificationRepository,
	resetRepo *repository.PasswordResetRepository,
	roleRepo *repository.RoleRepository,
	cfg *config.Config,
	rdb *redis.Client,
	mailer *email.Sender,
	db *gorm.DB,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		orgRepo:         orgRepo,
		sessionRepo:     sessionRepo,
		emailVerifyRepo: emailVerifyRepo,
		resetRepo:       resetRepo,
		roleRepo:        roleRepo,
		cfg:             cfg,
		rdb:             rdb,
		mailer:          mailer,
		cipher:          security.NewCredentialCipher(cfg.CredentialKey(), cfg.CredentialsEncryptionKeyPrevious...),
		keyring:         security.NewJWTKeyring(cfg.JWTSecret, cfg.JWTSecretPrevious...),
		db:              db,
	}
}

// auditAuth is a thin wrapper around middleware.LogSecurityEvent for
// the auth events this service is the sole source of truth for (login,
// logout, MFA, password changes) — none of them pass through
// AuditLog/RouteGuard's fiber.Ctx-based logging, since /auth/* mostly
// lives outside the RBAC-guarded route table entirely, and login in
// particular must be audited even when it fails (AuditLog only ever
// logs 2xx responses).
func (s *AuthService) auditAuth(orgID uuid.UUID, userID *uuid.UUID, eventType, subject, ip string, metadata map[string]string) {
	middleware.LogSecurityEvent(s.db, orgID, userID, "auth", eventType, subject, ip, metadata)
}

// signJWT signs claims with the current keyring key and tags the token
// with its kid, so a future rotation can still verify it via
// JWTSecretPrevious without invalidating tokens issued under this key.
func (s *AuthService) signJWT(claims jwt.MapClaims) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["kid"] = s.keyring.CurrentKeyID()
	return tok.SignedString(s.keyring.CurrentKey())
}

// parseJWT verifies a token against the keyring, resolving whichever key
// (current or retired) its kid header names.
func (s *AuthService) parseJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		kid, _ := t.Header["kid"].(string)
		key, ok := s.keyring.Key(kid)
		if !ok {
			return nil, errors.New("unknown signing key")
		}
		return key, nil
	})
}

// generateSecureToken returns a URL-safe random token plus the sha256 hash
// that should be persisted instead of the token itself — mirrors the
// pattern used for credential encryption: never store the thing that
// grants access, only something you can verify a presented value against.
func generateSecureToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])
	return token, hash, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// sendAuthEmail best-effort sends a transactional auth email. A delivery
// failure (or no provider configured, which is normal in dev) must never
// fail the calling request — the token is still valid and usable, it just
// wasn't emailed.
func (s *AuthService) sendAuthEmail(ctx context.Context, msg *email.Message) {
	if s.mailer == nil {
		return
	}
	if _, err := s.mailer.Send(ctx, msg); err != nil {
		log.Printf("auth: failed to send %s email to %v: %v", msg.Tags["category"], msg.To, err)
	}
}

type RegisterInput struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	OrgName   string `json:"org_name" validate:"required"`
	OrgSlug   string `json:"org_slug" validate:"required"`
	// OrgPassword becomes the shared secret every future teammate needs
	// to join this organization at signup (see JoinOrganization) or to
	// re-authorize identity-changing settings changes later. Required
	// here — Register only ever runs for a brand-new organization (see
	// LookupOrganization for how the frontend decides Register vs Join).
	OrgPassword string `json:"org_password" validate:"required,min=8"`
}

// OrgLookupResult tells the signup flow whether the organization the user
// typed already exists, so the frontend can branch between "create it"
// (Register, prompting for a new org password) and "join it" (Join,
// prompting for the existing org password) before any account is
// created. Deliberately exposes nothing besides existence + display
// name/slug — no member count, no plan, nothing that helps enumerate or
// profile an organization anonymously.
type OrgLookupResult struct {
	Exists bool   `json:"exists"`
	Name   string `json:"name,omitempty"`
	Slug   string `json:"slug,omitempty"`
}

// LookupOrganization resolves a user-typed organization name (or slug) to
// whether it already exists. Accepts a raw name so the frontend can call
// this straight from the "Organization name" field without deriving a
// slug client-side first.
func (s *AuthService) LookupOrganization(ctx context.Context, nameOrSlug string) (*OrgLookupResult, error) {
	slug := normalize.Slug(nameOrSlug)
	if slug == "" {
		return &OrgLookupResult{Exists: false}, nil
	}
	org, err := s.orgRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &OrgLookupResult{Exists: false}, nil
		}
		return nil, err
	}
	return &OrgLookupResult{Exists: true, Name: org.Name, Slug: org.Slug}, nil
}

// JoinInput is CASE 2 of signup (see LookupOrganization): the
// organization already exists, and the caller is proving they know its
// shared password rather than creating a new one.
type JoinInput struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	FirstName   string `json:"first_name" validate:"required"`
	LastName    string `json:"last_name" validate:"required"`
	OrgSlug     string `json:"org_slug" validate:"required"`
	OrgPassword string `json:"org_password" validate:"required"`
}

type LoginInput struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput, meta SessionMeta) (*models.User, *AuthTokens, error) {
	normalizedEmail := normalize.Email(input.Email)

	existing, _ := s.userRepo.FindByEmail(ctx, normalizedEmail)
	if existing != nil {
		return nil, nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	orgPasswordHash, err := bcrypt.GenerateFromPassword([]byte(input.OrgPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	var user *models.User
	var org *models.Organization

	// Organization creation, user creation, and owner-role provisioning
	// must succeed or fail together — a partial write here would leave
	// either an orphaned organization with no owner, or a user with no
	// role who (with RBAC enforced on every route) is locked out of the
	// account they just created.
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userRepoTx := repository.NewUserRepository(tx)
		roleRepoTx := repository.NewRoleRepository(tx)

		var txErr error
		org, txErr = createOrgWithUniqueSlug(ctx, tx, input.OrgName, input.OrgSlug, string(orgPasswordHash))
		if txErr != nil {
			return txErr
		}

		hashStr := string(hash)
		user = &models.User{
			OrganizationID: org.ID,
			Email:          normalizedEmail,
			PasswordHash:   &hashStr,
			FirstName:      input.FirstName,
			LastName:       input.LastName,
			Status:         "active",
			EmailVerified:  false,
		}
		if err := userRepoTx.Create(ctx, user); err != nil {
			return err
		}

		ownerRoleID, err := roleRepoTx.SeedDefaultRoles(ctx, org.ID)
		if err != nil {
			return fmt.Errorf("failed to provision organization roles: %w", err)
		}
		if err := roleRepoTx.AssignRole(ctx, user.ID, ownerRoleID); err != nil {
			return fmt.Errorf("failed to assign owner role: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	s.issueEmailVerification(ctx, user)

	tokens, err := s.generateTokens(ctx, user, meta)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// ErrOrgPasswordLocked mirrors ErrAccountLocked for the organization's
// shared join/settings password.
var ErrOrgPasswordLocked = errors.New("too many incorrect organization password attempts, try again later")

// JoinOrganization is CASE 2 of signup: the organization already exists
// (per a prior LookupOrganization call), and the caller is proving they
// know its shared password rather than minting a new organization. The
// joiner becomes a Member — Owner/Admin are never handed out just for
// knowing the org password; promoting someone requires an existing
// Owner/Admin acting through the team-management endpoints.
func (s *AuthService) JoinOrganization(ctx context.Context, input JoinInput, meta SessionMeta) (*models.User, *AuthTokens, error) {
	normalizedEmail := normalize.Email(input.Email)

	existing, _ := s.userRepo.FindByEmail(ctx, normalizedEmail)
	if existing != nil {
		return nil, nil, errors.New("email already registered")
	}

	org, err := s.orgRepo.FindBySlug(ctx, normalize.Slug(input.OrgSlug))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("organization not found")
		}
		return nil, nil, err
	}

	if err := s.verifyOrgPassword(ctx, org, input.OrgPassword, meta.IP); err != nil {
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	var user *models.User
	// User creation and role assignment must succeed together — same
	// reasoning as Register: a user with zero roles is locked out of the
	// org they just joined by RBAC itself.
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userRepoTx := repository.NewUserRepository(tx)
		roleRepoTx := repository.NewRoleRepository(tx)

		hashStr := string(hash)
		user = &models.User{
			OrganizationID: org.ID,
			Email:          normalizedEmail,
			PasswordHash:   &hashStr,
			FirstName:      input.FirstName,
			LastName:       input.LastName,
			Status:         "active",
			EmailVerified:  false,
		}
		if err := userRepoTx.Create(ctx, user); err != nil {
			return err
		}

		memberRole, err := roleRepoTx.FindByName(ctx, org.ID, "Member")
		if err != nil {
			return fmt.Errorf("failed to resolve default role: %w", err)
		}
		if err := roleRepoTx.AssignRole(ctx, user.ID, memberRole.ID); err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	s.issueEmailVerification(ctx, user)
	s.auditAuth(org.ID, &user.ID, "user_joined", user.Email+" joined the organization", meta.IP, nil)

	tokens, err := s.generateTokens(ctx, user, meta)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// verifyOrgPassword checks a presented organization password against the
// stored hash, enforcing the same brute-force lockout pattern as user
// login (see recordFailedLogin) but scoped to the organization as a
// whole: the counter and lock apply to every joiner or settings-change
// attempt against this org, not to one specific user, since the secret
// being guessed is shared.
func (s *AuthService) verifyOrgPassword(ctx context.Context, org *models.Organization, password, ip string) error {
	if org.PasswordLockedUntil != nil && time.Now().Before(*org.PasswordLockedUntil) {
		s.auditAuth(org.ID, nil, "org_password_blocked", "organization password attempt while locked", ip, nil)
		return ErrOrgPasswordLocked
	}

	if org.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*org.PasswordHash), []byte(password)) != nil {
		s.recordFailedOrgPassword(ctx, org, ip)
		return errors.New("incorrect organization password")
	}

	if org.FailedPasswordAttempts > 0 || org.PasswordLockedUntil != nil {
		org.FailedPasswordAttempts = 0
		org.PasswordLockedUntil = nil
		if err := s.orgRepo.Update(ctx, org); err != nil {
			log.Printf("auth: failed to reset org password attempt counter for org %s: %v", org.ID, err)
		}
	}
	return nil
}

func (s *AuthService) recordFailedOrgPassword(ctx context.Context, org *models.Organization, ip string) {
	org.FailedPasswordAttempts++
	locked := false
	if org.FailedPasswordAttempts >= s.cfg.MaxFailedLogins {
		until := time.Now().Add(s.cfg.LoginLockoutDuration)
		org.PasswordLockedUntil = &until
		locked = true
	}
	s.auditAuth(org.ID, nil, "org_password_failure", "failed organization password attempt", ip, map[string]string{
		"failed_count":            strconv.Itoa(org.FailedPasswordAttempts),
		"organization_now_locked": strconv.FormatBool(locked),
	})
	if err := s.orgRepo.Update(ctx, org); err != nil {
		log.Printf("auth: failed to persist failed org-password count for org %s: %v", org.ID, err)
	}
}

// createOrgWithUniqueSlug inserts a new organization, retrying with a
// randomized suffix whenever the chosen slug collides. Two different
// organizations legitimately named "Acme" (or any client racing another
// signup for the exact same name) must never surface a raw SQL error —
// they should each simply get a working, unique slug ("acme",
// "acme-4f2a", ...).
//
// A SELECT-then-INSERT existence check alone can't close the race: two
// concurrent transactions can both pass the check for the same candidate
// before either commits. So this instead attempts the INSERT directly
// and only reacts to an actual unique-constraint violation from Postgres
// — the one thing that's atomic and authoritative — retrying with a new
// candidate when that happens, rather than trusting a prior read.
//
// Each attempt is wrapped in its own SAVEPOINT. Postgres aborts the
// entire enclosing transaction after any failed statement — including an
// expected unique-violation — and refuses every further command on it
// ("current transaction is aborted", SQLSTATE 25P02) until a rollback
// happens. Without a savepoint to roll back to, the retry loop's next
// INSERT attempt would itself fail with that abort error instead of a
// clean shot at the next candidate slug, which is exactly what happened
// under concurrent signups for the same organization name before this
// was added: the first collision correctly retried, but every retry
// after it failed with 25P02 and the whole registration 500'd.
func createOrgWithUniqueSlug(ctx context.Context, tx *gorm.DB, name, requestedSlug, passwordHash string) (*models.Organization, error) {
	orgRepo := repository.NewOrganizationRepository(tx)
	base := normalize.Slug(requestedSlug)
	if base == "" {
		base = normalize.Slug(name)
	}
	if base == "" {
		base = "org"
	}

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 0 {
			suffix, err := randomSlugSuffix()
			if err != nil {
				return nil, err
			}
			slug = base + "-" + suffix
		}

		savepoint := fmt.Sprintf("org_slug_attempt_%d", attempt)
		if err := tx.SavePoint(savepoint).Error; err != nil {
			return nil, fmt.Errorf("failed to set savepoint: %w", err)
		}

		org := &models.Organization{Name: name, Slug: slug, Plan: "free", PasswordHash: &passwordHash}
		err := orgRepo.Create(ctx, org)
		if err == nil {
			return org, nil
		}
		// Matches any unique violation, not just a specific constraint
		// name: GORM's AutoMigrate (the actual schema mechanism this app
		// boots with — see internal/database/migrate.go) names a
		// uniqueIndex tag "idx_<table>_<column>", but a hand-run copy of
		// the documented .sql migrations would instead get Postgres's
		// own default "organizations_slug_key" from the column-level
		// UNIQUE constraint. The org insert above is the only thing in
		// this function that could hit a unique constraint at all, so
		// there's no ambiguity in treating any violation here as "slug
		// taken, retry" — hardcoding one specific name risked silently
		// turning every collision into a hard failure if the schema was
		// ever provisioned the other way.
		if !repository.IsUniqueViolation(err, "") {
			return nil, fmt.Errorf("failed to create organization: %w", err)
		}
		// Slug taken by a concurrent or prior signup — roll back to just
		// before the failed INSERT (undoing the abort, not the whole
		// transaction) and retry with a new random suffix.
		if rbErr := tx.RollbackTo(savepoint).Error; rbErr != nil {
			return nil, fmt.Errorf("failed to roll back to savepoint: %w", rbErr)
		}
	}
	return nil, errors.New("could not generate a unique organization slug, please try again")
}

// randomSlugSuffix returns a short random hex string for disambiguating
// a collided organization slug.
func randomSlugSuffix() (string, error) {
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// issueEmailVerification creates a fresh verification token (invalidating
// any previously issued ones) and emails it. Registration must still
// succeed even if this fails, so errors are logged, not returned.
func (s *AuthService) issueEmailVerification(ctx context.Context, user *models.User) {
	_ = s.emailVerifyRepo.InvalidateAllForUser(ctx, user.ID)

	token, hash, err := generateSecureToken()
	if err != nil {
		log.Printf("auth: failed to generate verification token for user %s: %v", user.ID, err)
		return
	}

	record := &models.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.cfg.EmailVerificationTTL),
	}
	if err := s.emailVerifyRepo.Create(ctx, record); err != nil {
		log.Printf("auth: failed to persist verification token for user %s: %v", user.ID, err)
		return
	}

	verifyURL := s.cfg.FrontendURL + "/verify-email?token=" + token
	s.sendAuthEmail(ctx, email.VerificationEmail(user.Email, s.cfg.SMTPFrom, s.cfg.SMTPFromName, verifyURL))
}

// VerifyEmail redeems a verification token. Tokens are single-use and
// expiring; an invalid/expired/already-used token all return the same
// generic error so the endpoint can't be used to enumerate valid tokens.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	record, err := s.emailVerifyRepo.FindByTokenHash(ctx, hashToken(token))
	if err != nil {
		return errors.New("invalid or expired verification token")
	}
	if record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		return errors.New("invalid or expired verification token")
	}

	user, err := s.userRepo.FindByID(ctx, record.UserID)
	if err != nil {
		return errors.New("invalid or expired verification token")
	}

	if err := s.emailVerifyRepo.MarkUsed(ctx, record.ID); err != nil {
		return err
	}

	now := time.Now()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	return s.userRepo.Update(ctx, user)
}

// ResendVerification reissues a verification token for the given email.
// Always returns nil regardless of whether the address exists or is
// already verified, so this endpoint can't be used to enumerate accounts.
func (s *AuthService) ResendVerification(ctx context.Context, emailAddr string) error {
	user, err := s.userRepo.FindByEmail(ctx, normalize.Email(emailAddr))
	if err != nil || user.EmailVerified {
		return nil
	}
	s.issueEmailVerification(ctx, user)
	return nil
}

// ErrAccountLocked is returned by Login when the account is currently
// locked out from too many consecutive failed attempts.
var ErrAccountLocked = errors.New("account is temporarily locked due to too many failed login attempts")

// ErrMFARequired is returned by Login when the password check passed but
// the account has TOTP enabled — the caller must complete
// VerifyMFALogin with a code before tokens are issued.
var ErrMFARequired = errors.New("mfa verification required")

func (s *AuthService) Login(ctx context.Context, input LoginInput, meta SessionMeta) (*models.User, *AuthTokens, error) {
	user, err := s.userRepo.FindByEmail(ctx, normalize.Email(input.Email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, err
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		s.auditAuth(user.OrganizationID, &user.ID, "login_blocked", "login attempt while account locked", meta.IP, nil)
		return nil, nil, ErrAccountLocked
	}

	if user.PasswordHash == nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		s.recordFailedLogin(ctx, user, meta.IP, "wrong password")
		return nil, nil, errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, nil, errors.New("account is not active")
	}

	if user.MFAEnabled {
		// Password verified but MFA still required: don't reset the
		// failed-login counter or issue tokens yet. The caller must call
		// VerifyMFALogin next with a short-lived pending ticket, which
		// itself is proof the password step already passed — a code
		// alone is never sufficient to complete login.
		return user, nil, ErrMFARequired
	}

	return s.completeLogin(ctx, user, meta)
}

// mfaPendingTTL bounds how long a password-verified-but-not-yet-MFA'd
// login attempt stays valid, limiting the window an intercepted ticket
// could be replayed in.
const mfaPendingTTL = 5 * time.Minute

// IssueMFAPendingTicket mints a short-lived, purpose-scoped JWT proving
// the password step of login already succeeded for this user. Handed to
// the client instead of a raw user ID so completing MFA can't be done by
// anyone who merely knows (or guesses) a user ID.
func (s *AuthService) IssueMFAPendingTicket(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"type": "mfa_pending",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(mfaPendingTTL).Unix(),
	}
	return s.signJWT(claims)
}

func (s *AuthService) parseMFAPendingTicket(ticket string) (uuid.UUID, error) {
	tok, err := s.parseJWT(ticket)
	if err != nil || !tok.Valid {
		return uuid.Nil, errors.New("invalid or expired mfa ticket")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "mfa_pending" {
		return uuid.Nil, errors.New("invalid or expired mfa ticket")
	}
	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return uuid.Nil, errors.New("invalid or expired mfa ticket")
	}
	return userID, nil
}

// completeLogin resets lockout state, updates LastLoginAt, and issues
// tokens. Split out of Login so VerifyMFALogin can reuse it once the TOTP
// step passes.
func (s *AuthService) completeLogin(ctx context.Context, user *models.User, meta SessionMeta) (*models.User, *AuthTokens, error) {
	tokens, err := s.generateTokens(ctx, user, meta)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	_ = s.userRepo.Update(ctx, user)

	s.auditAuth(user.OrganizationID, &user.ID, "login_success", "signed in", meta.IP, map[string]string{
		"remember_me": strconv.FormatBool(meta.RememberMe),
	})

	return user, tokens, nil
}

// recordFailedLogin increments the failure counter and locks the account
// once it crosses cfg.MaxFailedLogins. Errors updating the counter are
// logged, not returned — a persistence hiccup here must not turn into
// "you're locked out forever" nor silently disable lockout. reason is
// purely descriptive (audit metadata) — wrong password vs a failed MFA
// code both count toward the same lockout counter.
func (s *AuthService) recordFailedLogin(ctx context.Context, user *models.User, ip, reason string) {
	user.FailedLoginCount++
	locked := false
	if user.FailedLoginCount >= s.cfg.MaxFailedLogins {
		until := time.Now().Add(s.cfg.LoginLockoutDuration)
		user.LockedUntil = &until
		locked = true
	}
	s.auditAuth(user.OrganizationID, &user.ID, "login_failure", "failed login: "+reason, ip, map[string]string{
		"reason":             reason,
		"failed_count":       strconv.Itoa(user.FailedLoginCount),
		"account_now_locked": strconv.FormatBool(locked),
	})
	if err := s.userRepo.Update(ctx, user); err != nil {
		log.Printf("auth: failed to persist failed-login count for user %s: %v", user.ID, err)
	}
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	token, err := s.parseJWT(refreshToken)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if claims["type"] != "refresh" {
		return nil, errors.New("not a refresh token")
	}

	blacklisted, _ := s.rdb.Get(ctx, "blacklist:"+refreshToken).Result()
	if blacklisted != "" {
		return nil, errors.New("token has been revoked")
	}

	userIDStr, _ := claims["sub"].(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id in token")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// The session row is the source of truth for revocation (both
	// explicit "log out this device" and "log out everywhere"); the
	// Redis blacklist above is a fast-path cache of the same fact.
	session, err := s.sessionRepo.FindByTokenHash(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, errors.New("session not found")
	}
	if session.RevokedAt != nil {
		return nil, errors.New("token has been revoked")
	}

	meta := SessionMeta{RememberMe: session.RememberMe}
	if session.IP != nil {
		meta.IP = *session.IP
	}
	if session.UserAgent != nil {
		meta.UserAgent = *session.UserAgent
	}

	// Rotate: revoke the old session/token before issuing the new pair so
	// a stolen refresh token can't be replayed after a legitimate refresh.
	if err := s.sessionRepo.Revoke(ctx, session.ID); err != nil {
		log.Printf("auth: failed to revoke rotated session %s: %v", session.ID, err)
	}
	exp, _ := claims.GetExpirationTime()
	if exp != nil {
		s.rdb.Set(ctx, "blacklist:"+refreshToken, "1", time.Until(exp.Time))
	}

	return s.generateTokens(ctx, user, meta)
}

// ForgotPassword issues a password-reset token and emails it. Always
// returns nil (like ResendVerification) to avoid revealing whether the
// address has an account.
func (s *AuthService) ForgotPassword(ctx context.Context, emailAddr, requestIP string) error {
	user, err := s.userRepo.FindByEmail(ctx, normalize.Email(emailAddr))
	if err != nil {
		return nil
	}

	_ = s.resetRepo.InvalidateAllForUser(ctx, user.ID)

	token, hash, err := generateSecureToken()
	if err != nil {
		log.Printf("auth: failed to generate reset token for user %s: %v", user.ID, err)
		return nil
	}

	record := &models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hash,
		IssuedIP:  &requestIP,
		ExpiresAt: time.Now().Add(s.cfg.PasswordResetTTL),
	}
	if err := s.resetRepo.Create(ctx, record); err != nil {
		log.Printf("auth: failed to persist reset token for user %s: %v", user.ID, err)
		return nil
	}

	resetURL := s.cfg.FrontendURL + "/reset-password?token=" + token
	s.sendAuthEmail(ctx, email.PasswordResetEmail(user.Email, s.cfg.SMTPFrom, s.cfg.SMTPFromName, resetURL))
	return nil
}

// ResetPassword redeems a reset token, sets a new password, and revokes
// every existing session (refresh token) for the user — a leaked password
// being reset shouldn't leave an attacker's session alive.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword, ip string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	record, err := s.resetRepo.FindByTokenHash(ctx, hashToken(token))
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	if record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		return errors.New("invalid or expired reset token")
	}

	user, err := s.userRepo.FindByID(ctx, record.UserID)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.resetRepo.MarkUsed(ctx, record.ID); err != nil {
		return err
	}

	hashStr := string(hash)
	user.PasswordHash = &hashStr
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if err := s.sessionRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		log.Printf("auth: failed to revoke sessions for user %s after password reset: %v", user.ID, err)
	}

	s.auditAuth(user.OrganizationID, &user.ID, "password_reset", "password reset via emailed token", ip, nil)
	s.sendAuthEmail(ctx, email.PasswordChangedEmail(user.Email, s.cfg.SMTPFrom, s.cfg.SMTPFromName))
	return nil
}

// ChangePassword is the "I know my current password, I want a new one"
// flow (as opposed to ResetPassword's emailed-token flow for "I forgot
// it"). Requires the current password as re-authentication, then does
// the same thing ResetPassword does on success — revoke every other
// session and send a notification email — for the same reason: a
// password change is exactly the moment to invalidate whatever session
// an attacker might already be holding, whether the user is changing
// it routinely or because they suspect exactly that.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword, ip string) error {
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user.PasswordHash == nil {
		return errors.New("no password set for this account")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	hashStr := string(hash)
	user.PasswordHash = &hashStr
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if err := s.sessionRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		log.Printf("auth: failed to revoke sessions for user %s after password change: %v", user.ID, err)
	}

	s.auditAuth(user.OrganizationID, &user.ID, "password_changed", "password changed", ip, nil)
	s.sendAuthEmail(ctx, email.PasswordChangedEmail(user.Email, s.cfg.SMTPFrom, s.cfg.SMTPFromName))
	return nil
}

// DisableMFA requires the current password as re-authentication before
// turning MFA off — otherwise a hijacked, already-authenticated session
// alone would be enough to strip account protection.
func (s *AuthService) DisableMFA(ctx context.Context, userID uuid.UUID, currentPassword, ip string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword)) != nil {
		return errors.New("invalid credentials")
	}

	user.MFAEnabled = false
	user.MFASecretEncrypted = nil
	user.MFABackupCodesHash = datatypes.JSON("[]")
	user.MFAEnrolledAt = nil
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.auditAuth(user.OrganizationID, &user.ID, "mfa_disabled", "MFA disabled", ip, nil)
	return nil
}

// VerifyMFALogin completes a login that Login() paused with ErrMFARequired.
// Accepts either a live TOTP code or a backup code (consumed on use).
func (s *AuthService) VerifyMFALogin(ctx context.Context, ticket, code string, meta SessionMeta) (*models.User, *AuthTokens, error) {
	userID, err := s.parseMFAPendingTicket(ticket)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, errors.New("invalid or expired mfa ticket")
	}
	if !user.MFAEnabled || user.MFASecretEncrypted == nil {
		return nil, nil, errors.New("mfa is not enabled for this account")
	}
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, nil, ErrAccountLocked
	}

	secret, err := s.cipher.Decrypt(*user.MFASecretEncrypted)
	if err != nil {
		return nil, nil, errors.New("could not verify mfa code")
	}

	if security.ValidateTOTP(secret, code) {
		return s.completeLogin(ctx, user, meta)
	}

	if s.consumeBackupCode(ctx, user, code) {
		return s.completeLogin(ctx, user, meta)
	}

	s.recordFailedLogin(ctx, user, meta.IP, "invalid mfa code")
	return nil, nil, errors.New("invalid verification code")
}

// consumeBackupCode checks code against the stored hashes and, on match,
// removes that hash so the code can't be reused.
func (s *AuthService) consumeBackupCode(ctx context.Context, user *models.User, code string) bool {
	var hashes []string
	if err := json.Unmarshal(user.MFABackupCodesHash, &hashes); err != nil {
		return false
	}

	for i, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(code)) == nil {
			remaining := append(hashes[:i:i], hashes[i+1:]...)
			remainingJSON, err := json.Marshal(remaining)
			if err != nil {
				return false
			}
			user.MFABackupCodesHash = datatypes.JSON(remainingJSON)
			if err := s.userRepo.Update(ctx, user); err != nil {
				log.Printf("auth: failed to persist backup-code consumption for user %s: %v", user.ID, err)
			}
			return true
		}
	}
	return false
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string, orgID, userID uuid.UUID, ip string) error {
	s.rdb.Set(ctx, "blacklist:"+refreshToken, "1", 7*24*time.Hour)

	if session, err := s.sessionRepo.FindByTokenHash(ctx, hashToken(refreshToken)); err == nil {
		_ = s.sessionRepo.Revoke(ctx, session.ID)
	}

	s.auditAuth(orgID, &userID, "logout", "signed out", ip, nil)
	return nil
}

// ListSessions returns the user's active (non-revoked, non-expired)
// sessions for a "your devices" settings view.
func (s *AuthService) ListSessions(ctx context.Context, userID uuid.UUID) ([]models.RefreshToken, error) {
	return s.sessionRepo.ListActiveByUser(ctx, userID)
}

// RevokeSession lets a user sign a single device out remotely. Verifies
// the session belongs to the requesting user so one account can't revoke
// another's session by guessing an ID.
func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID, orgID uuid.UUID, ip string) error {
	sessions, err := s.sessionRepo.ListActiveByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.ID == sessionID {
			if err := s.sessionRepo.Revoke(ctx, sessionID); err != nil {
				return err
			}
			s.auditAuth(orgID, &userID, "session_revoked", "session revoked", ip, map[string]string{"session_id": sessionID.String()})
			return nil
		}
	}
	return errors.New("session not found")
}

// LogoutAllSessions revokes every session for the user ("log out
// everywhere"), e.g. after the user notices unfamiliar activity.
func (s *AuthService) LogoutAllSessions(ctx context.Context, userID, orgID uuid.UUID, ip string) error {
	if err := s.sessionRepo.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	s.auditAuth(orgID, &userID, "logout_all", "logged out of all sessions", ip, nil)
	return nil
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// MFAEnrollment is returned once, at enrollment time. The plaintext
// secret and backup codes are never retrievable again — only their
// encrypted/hashed forms are persisted.
type MFAEnrollment struct {
	Secret          string   `json:"secret"`
	ProvisioningURI string   `json:"provisioning_uri"`
	BackupCodes     []string `json:"backup_codes"`
}

// EnrollMFA generates a new TOTP secret and backup codes and stores them
// (encrypted/hashed) against the user, but leaves MFAEnabled false until
// ConfirmMFA proves the user actually has the secret loaded in an
// authenticator app. Re-enrolling before confirming simply replaces the
// pending secret.
func (s *AuthService) EnrollMFA(ctx context.Context, userID uuid.UUID) (*MFAEnrollment, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	secret, err := security.GenerateTOTPSecret()
	if err != nil {
		return nil, err
	}
	encSecret, err := s.cipher.Encrypt(secret)
	if err != nil {
		return nil, err
	}

	codes, err := security.GenerateBackupCodes(10)
	if err != nil {
		return nil, err
	}
	hashedCodes := make([]string, len(codes))
	for i, code := range codes {
		h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedCodes[i] = string(h)
	}
	codesJSON, err := json.Marshal(hashedCodes)
	if err != nil {
		return nil, err
	}

	user.MFASecretEncrypted = &encSecret
	user.MFABackupCodesHash = datatypes.JSON(codesJSON)
	user.MFAEnabled = false // requires ConfirmMFA
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &MFAEnrollment{
		Secret:          secret,
		ProvisioningURI: security.TOTPProvisioningURI(s.cfg.MFAIssuer, user.Email, secret),
		BackupCodes:     codes,
	}, nil
}

// ConfirmMFA proves the user's authenticator app is correctly loaded with
// the pending secret before MFA actually gates future logins.
func (s *AuthService) ConfirmMFA(ctx context.Context, userID uuid.UUID, code, ip string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user.MFASecretEncrypted == nil {
		return errors.New("no pending mfa enrollment")
	}

	secret, err := s.cipher.Decrypt(*user.MFASecretEncrypted)
	if err != nil {
		return errors.New("could not verify mfa secret")
	}
	if !security.ValidateTOTP(secret, code) {
		return errors.New("invalid verification code")
	}

	now := time.Now()
	user.MFAEnabled = true
	user.MFAEnrolledAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.auditAuth(user.OrganizationID, &user.ID, "mfa_enabled", "MFA enabled", ip, nil)
	return nil
}

// SessionMeta describes the device/request a login or token refresh came
// from, persisted alongside the session so "your active sessions" and
// "log out everywhere" have something meaningful to show/revoke.
type SessionMeta struct {
	IP         string
	UserAgent  string
	RememberMe bool
}

// generateTokens issues an access/refresh JWT pair and persists a durable
// session row for the refresh token (hash only) so it can be listed,
// individually revoked, or bulk-revoked later — the Redis blacklist alone
// only ever answers "was this specific token revoked", not "what sessions
// does this user have".
func (s *AuthService) generateTokens(ctx context.Context, user *models.User, meta SessionMeta) (*AuthTokens, error) {
	now := time.Now()
	accessExp := now.Add(15 * time.Minute)

	accessClaims := jwt.MapClaims{
		"sub":    user.ID.String(),
		"org_id": user.OrganizationID.String(),
		"email":  user.Email,
		"iat":    now.Unix(),
		"exp":    accessExp.Unix(),
		"type":   "access",
	}

	accessStr, err := s.signJWT(accessClaims)
	if err != nil {
		return nil, err
	}

	refreshTTL := s.cfg.RefreshExpiry
	if meta.RememberMe {
		refreshTTL = s.cfg.RememberMeExpiry
	}
	refreshExp := now.Add(refreshTTL)
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"iat":  now.Unix(),
		"exp":  refreshExp.Unix(),
		"type": "refresh",
	}

	refreshStr, err := s.signJWT(refreshClaims)
	if err != nil {
		return nil, err
	}

	session := &models.RefreshToken{
		UserID:     user.ID,
		TokenHash:  hashToken(refreshStr),
		RememberMe: meta.RememberMe,
		LastUsedAt: now,
		ExpiresAt:  refreshExp,
	}
	if meta.IP != "" {
		session.IP = &meta.IP
	}
	if meta.UserAgent != "" {
		session.UserAgent = &meta.UserAgent
		label := deviceLabelFromUserAgent(meta.UserAgent)
		session.DeviceLabel = &label
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		log.Printf("auth: failed to persist session for user %s: %v", user.ID, err)
	}

	return &AuthTokens{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}

// deviceLabelFromUserAgent produces a short human-readable label from a
// raw User-Agent header for display in the sessions list. Deliberately
// crude (no external dependency) — good enough to tell devices apart,
// not a full UA parser.
func deviceLabelFromUserAgent(ua string) string {
	switch {
	case strings.Contains(ua, "iPhone"):
		return "iPhone"
	case strings.Contains(ua, "iPad"):
		return "iPad"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "Macintosh"):
		return "Mac"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return "Unknown device"
	}
}
