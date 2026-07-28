package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/email"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
)

type AuthService struct {
	userRepo        *repository.UserRepository
	orgRepo         *repository.OrganizationRepository
	sessionRepo     *repository.SessionRepository
	emailVerifyRepo *repository.EmailVerificationRepository
	resetRepo       *repository.PasswordResetRepository
	cfg             *config.Config
	rdb             *redis.Client
	mailer          *email.Sender
	cipher          *security.CredentialCipher
}

func NewAuthService(
	userRepo *repository.UserRepository,
	orgRepo *repository.OrganizationRepository,
	sessionRepo *repository.SessionRepository,
	emailVerifyRepo *repository.EmailVerificationRepository,
	resetRepo *repository.PasswordResetRepository,
	cfg *config.Config,
	rdb *redis.Client,
	mailer *email.Sender,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		orgRepo:         orgRepo,
		sessionRepo:     sessionRepo,
		emailVerifyRepo: emailVerifyRepo,
		resetRepo:       resetRepo,
		cfg:             cfg,
		rdb:             rdb,
		mailer:          mailer,
		cipher:          security.NewCredentialCipher(cfg.CredentialKey(), cfg.CredentialsEncryptionKeyPrevious...),
	}
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
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=8"`
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name" validate:"required"`
	OrgName      string `json:"org_name" validate:"required"`
	OrgSlug      string `json:"org_slug" validate:"required"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*models.User, *AuthTokens, error) {
	existing, _ := s.userRepo.FindByEmail(ctx, input.Email)
	if existing != nil {
		return nil, nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	org := &models.Organization{
		Name: input.OrgName,
		Slug: input.OrgSlug,
		Plan: "free",
	}
	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, nil, err
	}

	hashStr := string(hash)
	user := &models.User{
		OrganizationID: org.ID,
		Email:          input.Email,
		PasswordHash:   &hashStr,
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Status:         "active",
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*models.User, *AuthTokens, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, err
	}

	if user.PasswordHash == nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, nil, errors.New("account is not active")
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.Update(ctx, user)

	return user, tokens, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
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

	// Blacklist the old refresh token
	exp, _ := claims.GetExpirationTime()
	if exp != nil {
		s.rdb.Set(ctx, "blacklist:"+refreshToken, "1", time.Until(exp.Time))
	}

	return s.generateTokens(user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	s.rdb.Set(ctx, "blacklist:"+refreshToken, "1", 7*24*time.Hour)
	return nil
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *AuthService) generateTokens(user *models.User) (*AuthTokens, error) {
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

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	refreshExp := now.Add(7 * 24 * time.Hour)
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"iat":  now.Unix(),
		"exp":  refreshExp.Unix(),
		"type": "refresh",
	}

	refreshTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshTokenJWT.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}
