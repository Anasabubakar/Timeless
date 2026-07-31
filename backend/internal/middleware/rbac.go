package middleware

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
			return fiber.NewError(fiber.StatusUnauthorized, "Please log in to continue.")
		}

		orgID := GetOrgID(c)
		if orgID == uuid.Nil {
			return fiber.NewError(fiber.StatusForbidden, "We couldn't determine your organization. Try logging in again.")
		}

		userPerms, err := m.getUserPermissions(userID, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "We couldn't check your permissions right now. Please try again.")
		}

		if missing, ok := satisfiesAll(userPerms, permissions); !ok {
			m.logDenial(c, userID, orgID, missing)
			return fiber.NewError(fiber.StatusForbidden, permissionDeniedMessage(missing))
		}

		return c.Next()
	}
}

func (m *RBACMiddleware) RequireAny(permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Please log in to continue.")
		}

		orgID := GetOrgID(c)
		if orgID == uuid.Nil {
			return fiber.NewError(fiber.StatusForbidden, "We couldn't determine your organization. Try logging in again.")
		}

		userPerms, err := m.getUserPermissions(userID, orgID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "We couldn't check your permissions right now. Please try again.")
		}

		if !satisfiesAny(userPerms, permissions) {
			m.logDenial(c, userID, orgID, strings.Join(permissions, " or "))
			return fiber.NewError(fiber.StatusForbidden, "You don't have permission to do that. Ask an Owner or Admin in your organization to grant you access.")
		}

		return c.Next()
	}
}

// satisfiesAll reports whether userPerms grants every permission in
// required (or the "*" wildcard). On failure it also returns the first
// permission that was missing, for error messages/audit logging.
func satisfiesAll(userPerms, required []string) (missing string, ok bool) {
	if slices.Contains(userPerms, PermAll) {
		return "", true
	}
	for _, perm := range required {
		if !slices.Contains(userPerms, perm) {
			return perm, false
		}
	}
	return "", true
}

// satisfiesAny reports whether userPerms grants at least one permission
// in required (or the "*" wildcard). An empty required list is
// vacuously unsatisfied — RequireAny with no arguments is a
// configuration mistake, not an open door.
func satisfiesAny(userPerms, required []string) bool {
	if slices.Contains(userPerms, PermAll) {
		return true
	}
	for _, perm := range required {
		if slices.Contains(userPerms, perm) {
			return true
		}
	}
	return false
}

// logDenial records a permission-denied security event. Async (like
// AuditLog) so a slow/failed write never adds latency to — or fails — the
// request that's already being rejected; the denial itself was already
// decided by the time this runs.
func (m *RBACMiddleware) logDenial(c fiber.Ctx, userID, orgID uuid.UUID, requiredPermission string) {
	LogSecurityEvent(m.db, orgID, &userID, "authorization", "permission_denied", "denied: "+requiredPermission, c.IP(), map[string]string{
		"method":              c.Method(),
		"path":                c.Path(),
		"required_permission": requiredPermission,
	})
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
	return satisfiesAny(perms, []string{permission}), nil
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
