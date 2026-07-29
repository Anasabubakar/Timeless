package handler

import (
	"encoding/json"
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

// ownerRoleName must match the "Owner" system role seeded by
// RoleRepository.SeedDefaultRoles.
const ownerRoleName = "Owner"

type TeamHandler struct {
	db       *gorm.DB
	roleRepo *repository.RoleRepository
}

func NewTeamHandler(db *gorm.DB, roleRepo *repository.RoleRepository) *TeamHandler {
	return &TeamHandler{db: db, roleRepo: roleRepo}
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

func (h *TeamHandler) ListMembers(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

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
		Roles     []string  `json:"roles"`
		CreatedAt string    `json:"created_at"`
	}

	members := make([]MemberResponse, 0, len(users))
	for _, u := range users {
		roles := make([]string, 0, len(u.Roles))
		for _, r := range u.Roles {
			roles = append(roles, r.Name)
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

func (h *TeamHandler) InviteMember(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var input InviteMemberInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var existing models.User
	if err := h.db.Where("email = ? AND organization_id = ?", input.Email, orgID).First(&existing).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "user already exists in this organization")
	}

	user := models.User{
		OrganizationID: orgID,
		Email:          input.Email,
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Status:         "invited",
	}
	user.ID = uuid.New()

	if err := h.db.Create(&user).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create user")
	}

	var role models.Role
	if err := h.db.Where("organization_id = ? AND name = ?", orgID, input.Role).First(&role).Error; err == nil {
		h.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING", user.ID, role.ID)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": user})
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
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
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

	h.db.Exec("DELETE FROM user_roles WHERE user_id = ?", memberID)

	for _, roleName := range input.Roles {
		var role models.Role
		if err := h.db.Where("organization_id = ? AND name = ?", orgID, roleName).First(&role).Error; err == nil {
			h.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", memberID, role.ID)
		}
	}

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
