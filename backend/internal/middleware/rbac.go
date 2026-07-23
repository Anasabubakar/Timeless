package middleware

import (
	"encoding/json"
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type RBACMiddleware struct {
	db *gorm.DB
}

func NewRBAC(db *gorm.DB) *RBACMiddleware {
	return &RBACMiddleware{db: db}
}

func (m *RBACMiddleware) Require(permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		orgID := GetOrgID(c)
		if orgID == uuid.Nil {
			return fiber.NewError(fiber.StatusForbidden, "organization context required")
		}

		userPerms, err := m.getUserPermissions(userID, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to resolve permissions")
		}

		if slices.Contains(userPerms, "*") {
			return c.Next()
		}

		for _, required := range permissions {
			if !slices.Contains(userPerms, required) {
				return fiber.NewError(fiber.StatusForbidden, "insufficient permissions: "+required)
			}
		}

		return c.Next()
	}
}

func (m *RBACMiddleware) RequireAny(permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		orgID := GetOrgID(c)
		if orgID == uuid.Nil {
			return fiber.NewError(fiber.StatusForbidden, "organization context required")
		}

		userPerms, err := m.getUserPermissions(userID, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to resolve permissions")
		}

		if slices.Contains(userPerms, "*") {
			return c.Next()
		}

		for _, required := range permissions {
			if slices.Contains(userPerms, required) {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}

func (m *RBACMiddleware) getUserPermissions(userID, orgID uuid.UUID) ([]string, error) {
	var roles []models.Role
	err := m.db.
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.organization_id = ?", userID, orgID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}

	var allPerms []string
	for _, role := range roles {
		var perms []string
		if err := json.Unmarshal(role.Permissions, &perms); err != nil {
			continue
		}
		allPerms = append(allPerms, perms...)
	}

	return allPerms, nil
}
