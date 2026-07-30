package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/email"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
	"github.com/timeless/backend/internal/repository"
)

// invitationTTL bounds how long an invitation link stays acceptable —
// long enough that a real person checking email over a normal work week
// won't hit it, short enough that a link sitting unused in an old email
// thread eventually stops being a standing way into the organization.
const invitationTTL = 7 * 24 * time.Hour

// ErrAlreadyMember is returned when the invited email already belongs to
// a user in this organization.
var ErrAlreadyMember = errors.New("user already exists in this organization")

// ErrAlreadyInvited is returned when the invited email already has an
// active (pending, unexpired) invitation to this organization.
var ErrAlreadyInvited = errors.New("this email already has a pending invitation")

// ErrInvitationInvalid covers every reason an accept attempt can't
// proceed: unknown token, already accepted, revoked, or expired — folded
// into one message rather than distinguished, so a stale link can't be
// used to probe which of those states it's in.
var ErrInvitationInvalid = errors.New("this invitation link is invalid or has expired")

type InvitationService struct {
	repo     *repository.InvitationRepository
	roleRepo *repository.RoleRepository
	userRepo *repository.UserRepository
	orgRepo  *repository.OrganizationRepository
	authSvc  *AuthService
	mailer   *email.Sender
	db       *gorm.DB
}

func NewInvitationService(
	repo *repository.InvitationRepository,
	roleRepo *repository.RoleRepository,
	userRepo *repository.UserRepository,
	orgRepo *repository.OrganizationRepository,
	authSvc *AuthService,
	mailer *email.Sender,
	db *gorm.DB,
) *InvitationService {
	return &InvitationService{
		repo: repo, roleRepo: roleRepo, userRepo: userRepo, orgRepo: orgRepo,
		authSvc: authSvc, mailer: mailer, db: db,
	}
}

func (s *InvitationService) audit(orgID uuid.UUID, actorID *uuid.UUID, eventType, subject, ip string, metadata map[string]string) {
	middleware.LogSecurityEvent(s.db, orgID, actorID, "invitation", eventType, subject, ip, metadata)
}

// Create invites an email address to join an organization at a given
// role. Returns the invitation and the raw (unhashed) token — the only
// place that value ever exists outside the invitee's inbox, since only
// TokenHash is persisted.
func (s *InvitationService) Create(ctx context.Context, orgID, invitedByID uuid.UUID, emailAddr, roleName, ip string) (*models.Invitation, string, error) {
	normalizedEmail := normalize.Email(emailAddr)

	if existing, _ := s.userRepo.FindByOrgAndEmail(ctx, orgID, normalizedEmail); existing != nil {
		return nil, "", ErrAlreadyMember
	}
	if existing, _ := s.repo.FindActiveByEmail(ctx, orgID, normalizedEmail); existing != nil {
		return nil, "", ErrAlreadyInvited
	}

	if _, err := s.roleRepo.FindByName(ctx, orgID, roleName); err != nil {
		return nil, "", fmt.Errorf("unknown role: %s", roleName)
	}

	rawToken, hash, err := generateSecureToken()
	if err != nil {
		return nil, "", err
	}

	inv := &models.Invitation{
		OrganizationID: orgID,
		Email:          normalizedEmail,
		Role:           roleName,
		TokenHash:      hash,
		InvitedByID:    invitedByID,
		ExpiresAt:      time.Now().Add(invitationTTL),
	}
	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, "", err
	}

	s.audit(orgID, &invitedByID, "invitation_created", "invited "+normalizedEmail+" as "+roleName, ip, map[string]string{
		"invitation_id": inv.ID.String(),
		"role":          roleName,
	})

	s.sendInvitationEmail(ctx, orgID, normalizedEmail, rawToken)

	return inv, rawToken, nil
}

// sendInvitationEmail best-effort emails the accept link — like
// sendAuthEmail, a delivery failure must never fail the invite itself;
// the token is valid and usable, an Owner/Admin can just re-share the
// link some other way.
func (s *InvitationService) sendInvitationEmail(ctx context.Context, orgID uuid.UUID, toEmail, rawToken string) {
	if s.mailer == nil {
		return
	}
	org, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return
	}
	acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.authSvc.cfg.FrontendURL, rawToken)
	msg := email.InvitationEmail(toEmail, org.Name, acceptURL, s.authSvc.cfg.SMTPFrom, s.authSvc.cfg.SMTPFromName)
	if _, err := s.mailer.Send(ctx, msg); err != nil {
		fmt.Printf("invitation: failed to send invite email to %s: %v\n", toEmail, err)
	}
}

// ListPending returns every not-yet-accepted, not-yet-revoked,
// not-yet-expired invitation for an organization.
func (s *InvitationService) ListPending(ctx context.Context, orgID uuid.UUID) ([]models.Invitation, error) {
	return s.repo.ListPending(ctx, orgID)
}

// Revoke cancels a pending invitation before it's accepted.
func (s *InvitationService) Revoke(ctx context.Context, orgID, invitationID, actorID uuid.UUID, ip string) error {
	inv, err := s.repo.FindByID(ctx, invitationID, orgID)
	if err != nil {
		return errors.New("invitation not found")
	}

	rows, err := s.repo.Revoke(ctx, invitationID, orgID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("invitation is no longer pending")
	}

	s.audit(orgID, &actorID, "invitation_revoked", "revoked invitation for "+inv.Email, ip, map[string]string{
		"invitation_id": invitationID.String(),
	})
	return nil
}

// AcceptInput is what an invited person submits to actually join.
type AcceptInput struct {
	Token     string `json:"token" validate:"required"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// Accept redeems an invitation token: creates the user account (which,
// unlike the old invite flow this replaces, never existed until this
// moment — nothing to be locked out of before agreeing to join), assigns
// the invited role, marks the invitation accepted, and logs the new
// member straight in, matching Register/JoinOrganization's behavior.
func (s *InvitationService) Accept(ctx context.Context, input AcceptInput, meta SessionMeta) (*models.User, *AuthTokens, error) {
	inv, err := s.repo.FindPendingByTokenHash(ctx, hashToken(input.Token))
	if err != nil {
		return nil, nil, ErrInvitationInvalid
	}

	if existing, _ := s.userRepo.FindByEmail(ctx, inv.Email); existing != nil {
		return nil, nil, errors.New("email already registered")
	}

	role, err := s.roleRepo.FindByName(ctx, inv.OrganizationID, inv.Role)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve invited role: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	var user *models.User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userRepoTx := repository.NewUserRepository(tx)
		roleRepoTx := repository.NewRoleRepository(tx)

		hashStr := string(hash)
		user = &models.User{
			OrganizationID: inv.OrganizationID,
			Email:          inv.Email,
			PasswordHash:   &hashStr,
			FirstName:      input.FirstName,
			LastName:       input.LastName,
			Status:         "active",
			EmailVerified:  true, // accepting via a mailed link already proves inbox ownership
		}
		if err := userRepoTx.Create(ctx, user); err != nil {
			return err
		}
		return roleRepoTx.AssignRole(ctx, user.ID, role.ID)
	})
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.MarkAccepted(ctx, inv.ID); err != nil {
		fmt.Printf("invitation: failed to mark %s accepted: %v\n", inv.ID, err)
	}

	if refreshed, refreshErr := s.userRepo.FindByID(ctx, user.ID); refreshErr == nil {
		user = refreshed
	}

	s.audit(inv.OrganizationID, &user.ID, "invitation_accepted", user.Email+" accepted their invitation", meta.IP, map[string]string{
		"invitation_id": inv.ID.String(),
	})

	tokens, err := s.authSvc.generateTokens(ctx, user, meta)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}
