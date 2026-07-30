package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
	"github.com/timeless/backend/internal/repository"
)

const ownerRoleName = "Owner"

// ErrNotOwner is returned when a caller without the Owner role attempts
// an Owner-only action (identity-changing settings, ownership transfer).
var ErrNotOwner = errors.New("only the organization's Owner can perform this action")

// ErrSlugTaken is returned when a requested slug change collides with
// another organization's existing slug.
var ErrSlugTaken = errors.New("that slug is already taken")

type OrganizationService struct {
	repo             *repository.OrganizationRepository
	roleRepo         *repository.RoleRepository
	userRepo         *repository.UserRepository
	orgPasswordGuard *orgPasswordGuard
	db               *gorm.DB
}

func NewOrganizationService(
	repo *repository.OrganizationRepository,
	roleRepo *repository.RoleRepository,
	userRepo *repository.UserRepository,
	cfg *config.Config,
	db *gorm.DB,
) *OrganizationService {
	return &OrganizationService{
		repo:             repo,
		roleRepo:         roleRepo,
		userRepo:         userRepo,
		orgPasswordGuard: newOrgPasswordGuard(cfg, repo, db),
		db:               db,
	}
}

func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	return s.repo.FindByID(ctx, id)
}

// Update applies non-identity settings (logo, domain, arbitrary settings
// JSON) with no additional authorization beyond the route's own RBAC
// permission check — these aren't the name/slug/password fields the
// organization-security requirements gate behind Owner + password
// reverification (see UpdateSecure).
func (s *OrganizationService) Update(ctx context.Context, org *models.Organization) error {
	return s.repo.Update(ctx, org)
}

// UpdateSecureInput carries the identity-changing fields — name, slug,
// and/or a new password — that require Owner + current password
// reverification. Fields left nil are left unchanged.
type UpdateSecureInput struct {
	Name        *string
	Slug        *string
	NewPassword *string
}

func (s *OrganizationService) audit(orgID uuid.UUID, actorID *uuid.UUID, eventType, subject, ip string, metadata map[string]string) {
	middleware.LogSecurityEvent(s.db, orgID, actorID, "organization", eventType, subject, ip, metadata)
}

// UpdateSecure renames the organization, changes its slug, and/or
// rotates its password — every field the spec requires Owner permission
// plus current-password reverification for. Not wrapped in a DB
// transaction: it's a single row update after validation, so there's
// nothing partial to roll back.
func (s *OrganizationService) UpdateSecure(ctx context.Context, orgID, actorID uuid.UUID, input UpdateSecureInput, currentPassword, ip string) (*models.Organization, error) {
	isOwner, err := s.roleRepo.HasRole(ctx, orgID, actorID, ownerRoleName)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !isOwner {
		return nil, ErrNotOwner
	}

	org, err := s.repo.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if err := s.orgPasswordGuard.Verify(ctx, org, currentPassword, ip); err != nil {
		return nil, err
	}

	var changed []string

	if input.Name != nil && strings.TrimSpace(*input.Name) != "" && *input.Name != org.Name {
		org.Name = strings.TrimSpace(*input.Name)
		changed = append(changed, "name")
	}

	if input.Slug != nil {
		newSlug := normalize.Slug(*input.Slug)
		if newSlug == "" {
			return nil, errors.New("invalid slug")
		}
		if newSlug != org.Slug {
			existing, err := s.repo.FindBySlug(ctx, newSlug)
			if err == nil && existing.ID != org.ID {
				return nil, ErrSlugTaken
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			org.Slug = newSlug
			changed = append(changed, "slug")
		}
	}

	if input.NewPassword != nil {
		if len(*input.NewPassword) < 8 {
			return nil, errors.New("new organization password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashStr := string(hash)
		org.PasswordHash = &hashStr
		changed = append(changed, "password")
	}

	if len(changed) == 0 {
		return org, nil
	}

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, err
	}

	s.audit(org.ID, &actorID, "org_settings_changed", "changed: "+strings.Join(changed, ", "), ip, map[string]string{
		"fields": strings.Join(changed, ","),
	})
	return org, nil
}

// TransferOwnership moves the Owner role from the current caller to
// another member of the same organization: grants Owner to the target,
// then demotes the caller to Admin (never left with zero roles) and
// strips their Owner role. Requires the caller to already be Owner and
// to reverify the organization password — the highest-privilege action
// this API exposes.
func (s *OrganizationService) TransferOwnership(ctx context.Context, orgID, actorID, targetUserID uuid.UUID, currentPassword, ip string) error {
	if actorID == targetUserID {
		return errors.New("already the owner")
	}

	isOwner, err := s.roleRepo.HasRole(ctx, orgID, actorID, ownerRoleName)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !isOwner {
		return ErrNotOwner
	}

	org, err := s.repo.FindByID(ctx, orgID)
	if err != nil {
		return err
	}
	if err := s.orgPasswordGuard.Verify(ctx, org, currentPassword, ip); err != nil {
		return err
	}

	target, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil || target.OrganizationID != orgID {
		return errors.New("target user not found in this organization")
	}

	ownerRole, err := s.roleRepo.FindByName(ctx, orgID, ownerRoleName)
	if err != nil {
		return fmt.Errorf("failed to resolve Owner role: %w", err)
	}
	adminRole, err := s.roleRepo.FindByName(ctx, orgID, "Admin")
	if err != nil {
		return fmt.Errorf("failed to resolve Admin role: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roleRepoTx := repository.NewRoleRepository(tx)
		if err := roleRepoTx.AssignRole(ctx, targetUserID, ownerRole.ID); err != nil {
			return err
		}
		if err := roleRepoTx.AssignRole(ctx, actorID, adminRole.ID); err != nil {
			return err
		}
		return roleRepoTx.RevokeRole(ctx, actorID, ownerRole.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to transfer ownership: %w", err)
	}

	s.audit(orgID, &actorID, "owner_transferred", "transferred ownership to "+target.Email, ip, map[string]string{
		"new_owner_id": targetUserID.String(),
	})
	return nil
}
