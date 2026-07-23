package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantMiddleware struct {
	db *gorm.DB
}

func NewTenant(db *gorm.DB) *TenantMiddleware {
	return &TenantMiddleware{db: db}
}

func (m *TenantMiddleware) Handle(c fiber.Ctx) error {
	orgIDStr, ok := c.Locals("org_id").(string)
	if !ok || orgIDStr == "" {
		return fiber.NewError(fiber.StatusForbidden, "organization context required")
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "invalid organization id")
	}

	c.Locals("org_uuid", orgID)
	return c.Next()
}

func GetOrgID(c fiber.Ctx) uuid.UUID {
	orgID, _ := c.Locals("org_uuid").(uuid.UUID)
	return orgID
}

func GetUserID(c fiber.Ctx) uuid.UUID {
	userIDStr, _ := c.Locals("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)
	return uid
}
