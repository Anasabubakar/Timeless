package middleware

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
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
				m.logDenial(c, userID, orgID, required)
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

		m.logDenial(c, userID, orgID, strings.Join(permissions, " or "))
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}

// logDenial records a permission-denied security event. Async (like
// AuditLog) so a slow/failed write never adds latency to — or fails — the
// request that's already being rejected; the denial itself was already
// decided by the time this runs.
func (m *RBACMiddleware) logDenial(c fiber.Ctx, userID, orgID uuid.UUID, requiredPermission string) {
	if m.db == nil {
		return
	}
	meta := map[string]string{
		"method":              c.Method(),
		"path":                c.Path(),
		"required_permission": requiredPermission,
	}
	metaJSON, _ := json.Marshal(meta)
	ip := c.IP()

	activity := models.Activity{
		OrganizationID: orgID,
		UserID:         &userID,
		EntityType:     "authorization",
		Type:           "permission_denied",
		Subject:        "denied: " + requiredPermission,
		IPAddress:      &ip,
		Metadata:       datatypes.JSON(metaJSON),
	}
	activity.ID = uuid.New()

	go m.db.Create(&activity)
}

// HasPermission is the non-middleware entry point into the same
// permission resolution Require/RequireAny use, for handlers that need
// to make a permission-gated decision inline (e.g. which fields to
// include in a response) rather than allow-or-403 an entire route.
func (m *RBACMiddleware) HasPermission(userID, orgID uuid.UUID, permission string) (bool, error) {
	perms, err := m.getUserPermissions(userID, orgID)
	if err != nil {
		return false, err
	}
	return slices.Contains(perms, "*") || slices.Contains(perms, permission), nil
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
