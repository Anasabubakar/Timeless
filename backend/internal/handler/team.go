package handler

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/service"
)

// ownerRoleName must match the "Owner" system role seeded by
// RoleRepository.SeedDefaultRoles.
const ownerRoleName = "Owner"

type TeamHandler struct {
	db       *gorm.DB
	roleRepo *repository.RoleRepository
	rbac     *middleware.RBACMiddleware
	invSvc   *service.InvitationService
}

func NewTeamHandler(db *gorm.DB, roleRepo *repository.RoleRepository, rbac *middleware.RBACMiddleware, invSvc *service.InvitationService) *TeamHandler {
	return &TeamHandler{db: db, roleRepo: roleRepo, rbac: rbac, invSvc: invSvc}
}

// hasRole reports whether user currently holds the named role.
func (h *TeamHandler) hasRole(user *models.User, roleName string) bool {
	for _, r := range user.Roles {
		if r.Name == roleName {
			return true
		}
	}
	return false
}

// requesterIsOwner reports whether the authenticated caller currently
// holds the Owner role. team:manage (granted to Admin and above) is
// enough to invite or promote Admins/Managers/Members/Guests, but
// granting Owner — the one tier with special removal/demotion
// protection — must require already being an Owner. Without this, any
// Admin could mint themselves (or an accomplice) a second Owner account
// with no additional check, which defeats the point of a distinct tier.
func (h *TeamHandler) requesterIsOwner(c fiber.Ctx, orgID uuid.UUID) (bool, error) {
	userID := middleware.GetUserID(c)
	var requester models.User
	if err := h.db.Preload("Roles", "organization_id = ?", orgID).
		Where("id = ? AND organization_id = ?", userID, orgID).First(&requester).Error; err != nil {
		return false, err
	}
	return h.hasRole(&requester, ownerRoleName), nil
}

func (h *TeamHandler) ListMembers(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	// Which specific roles a teammate holds is more sensitive than the
	// roster itself (it maps out who has elevated access — useful
	// reconnaissance for social engineering) — only include it for
	// callers who can actually manage team membership. Everyone with
	// team:read still sees the roster, just not each person's roles.
	canManage, err := h.rbac.HasPermission(userID, orgID, middleware.PermTeamManage)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to resolve permissions")
	}

	var users []models.User
	if err := h.db.
		Preload("Roles", "organization_id = ?", orgID).
		Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Find(&users).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list members")
	}

	type MemberResponse struct {
		ID        uuid.UUID `json:"id"`
		Email     string    `json:"email"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		AvatarURL *string   `json:"avatar_url,omitempty"`
		JobTitle  *string   `json:"job_title,omitempty"`
		Status    string    `json:"status"`
		Roles     []string  `json:"roles,omitempty"`
		CreatedAt string    `json:"created_at"`
	}

	members := make([]MemberResponse, 0, len(users))
	for _, u := range users {
		var roles []string
		if canManage {
			roles = make([]string, 0, len(u.Roles))
			for _, r := range u.Roles {
				roles = append(roles, r.Name)
			}
		}
		members = append(members, MemberResponse{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			AvatarURL: u.AvatarURL,
			JobTitle:  u.JobTitle,
			Status:    u.Status,
			Roles:     roles,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{"data": members})
}

type InviteMemberInput struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Role      string `json:"role" validate:"required"`
}

// InviteMember sends an actual invitation (token + email) rather than
// creating an account directly — see InvitationService.Create. The
// invited person doesn't exist as a User until they accept it.
func (h *TeamHandler) InviteMember(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	actorID := middleware.GetUserID(c)

	var input InviteMemberInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.Email = normalize.Email(input.Email)

	if input.Role == ownerRoleName {
		isOwner, err := h.requesterIsOwner(c, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to verify ownership")
		}
		if !isOwner {
			return fiber.NewError(fiber.StatusForbidden, "only an existing Owner can invite a new Owner")
		}
	}

	inv, _, err := h.invSvc.Create(c.Context(), orgID, actorID, input.Email, input.Role, c.IP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAlreadyMember):
			return fiber.NewError(fiber.StatusConflict, err.Error())
		case errors.Is(err, service.ErrAlreadyInvited):
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": inv})
}

// ListPendingInvitations: GET /team/invitations
func (h *TeamHandler) ListPendingInvitations(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	invitations, err := h.invSvc.ListPending(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list invitations")
	}
	return c.JSON(fiber.Map{"data": invitations})
}

// RevokeInvitation: DELETE /team/invitations/:id
func (h *TeamHandler) RevokeInvitation(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	actorID := middleware.GetUserID(c)
	invID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invitation id")
	}

	if err := h.invSvc.Revoke(c.Context(), orgID, invID, actorID, c.IP()); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

type UpdateMemberRoleInput struct {
	Roles []string `json:"roles" validate:"required"`
}

func (h *TeamHandler) UpdateMemberRole(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	memberID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	var input UpdateMemberRoleInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	var user models.User
	if err := h.db.Preload("Roles", "organization_id = ?", orgID).
		Where("id = ? AND organization_id = ?", memberID, orgID).First(&user).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "member not found")
	}

	if h.hasRole(&user, ownerRoleName) && !slices.Contains(input.Roles, ownerRoleName) {
		remaining, err := h.roleRepo.CountUsersWithRole(c.Context(), orgID, ownerRoleName, memberID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to verify ownership")
		}
		if remaining == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "cannot remove Owner from the organization's last owner")
		}
	}

	if !h.hasRole(&user, ownerRoleName) && slices.Contains(input.Roles, ownerRoleName) {
		isOwner, err := h.requesterIsOwner(c, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to verify ownership")
		}
		if !isOwner {
			return fiber.NewError(fiber.StatusForbidden, "only an existing Owner can grant the Owner role")
		}
	}

	previousRoles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		previousRoles = append(previousRoles, r.Name)
	}

	h.db.Exec("DELETE FROM user_roles WHERE user_id = ?", memberID)

	for _, roleName := range input.Roles {
		var role models.Role
		if err := h.db.Where("organization_id = ? AND name = ?", orgID, roleName).First(&role).Error; err == nil {
			h.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", memberID, role.ID)
		}
	}

	actorID := middleware.GetUserID(c)
	middleware.LogSecurityEvent(h.db, orgID, &actorID, "team", "role_changed",
		"changed roles for "+user.Email, c.IP(), map[string]string{
			"target_user_id": memberID.String(),
			"previous_roles": strings.Join(previousRoles, ","),
			"new_roles":      strings.Join(input.Roles, ","),
		})

	return c.JSON(fiber.Map{"message": "roles updated"})
}

func (h *TeamHandler) RemoveMember(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	memberID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	if memberID == userID {
		return fiber.NewError(fiber.StatusBadRequest, "cannot remove yourself")
	}

	var member models.User
	if err := h.db.Preload("Roles", "organization_id = ?", orgID).
		Where("id = ? AND organization_id = ?", memberID, orgID).First(&member).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "member not found")
	}

	if h.hasRole(&member, ownerRoleName) {
		remaining, err := h.roleRepo.CountUsersWithRole(c.Context(), orgID, ownerRoleName, memberID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to verify ownership")
		}
		if remaining == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "cannot remove the organization's last owner")
		}
	}

	result := h.db.Where("id = ? AND organization_id = ?", memberID, orgID).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "member not found")
	}

	middleware.LogSecurityEvent(h.db, orgID, &userID, "team", "member_removed",
		"removed "+member.Email, c.IP(), map[string]string{
			"removed_user_id": memberID.String(),
		})

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *TeamHandler) ListRoles(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var roles []models.Role
	if err := h.db.Where("organization_id = ?", orgID).Order("is_system DESC, name ASC").Find(&roles).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list roles")
	}

	type RoleResponse struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Permissions []string  `json:"permissions"`
		IsSystem    bool      `json:"is_system"`
	}

	resp := make([]RoleResponse, 0, len(roles))
	for _, r := range roles {
		var perms []string
		json.Unmarshal(r.Permissions, &perms)
		resp = append(resp, RoleResponse{
			ID:          r.ID,
			Name:        r.Name,
			Permissions: perms,
			IsSystem:    r.IsSystem,
		})
	}

	return c.JSON(fiber.Map{"data": resp})
}
