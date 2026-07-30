package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type AuditConfig struct {
	DB *gorm.DB
}

func AuditLog(cfg AuditConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()

		status := c.Response().StatusCode()
		method := c.Method()

		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			return err
		}

		// 401/403/429 already get their own more specific security event
		// (auth failure never reaches this middleware at all; RBAC denial
		// and rate-limit violations log themselves via LogSecurityEvent) —
		// logging them again here would just be a noisier duplicate. A 5xx
		// on a mutating request is different: nothing else records that a
		// write was *attempted* and failed unexpectedly, which is exactly
		// the kind of thing "data deletion"/"admin actions" audit coverage
		// is supposed to catch, not just successful ones.
		if status >= 400 && status < 500 {
			return err
		}

		orgID := GetOrgID(c)
		userID := GetUserID(c)
		if orgID == uuid.Nil {
			return err
		}

		path := c.Path()
		action := mapAction(method)
		if status >= 500 {
			action = "failed_" + action
		}
		entityType, entityID := parseEntity(path)

		if entityType == "" {
			return err
		}

		subject := fmt.Sprintf("%s %s", action, entityType)
		ip := c.IP()

		meta := map[string]interface{}{
			"method": method,
			"path":   path,
			"status": status,
		}
		metaJSON, _ := json.Marshal(meta)

		activity := models.Activity{
			OrganizationID: orgID,
			UserID:         &userID,
			EntityType:     entityType,
			EntityID:       entityIDOrNil(entityID),
			Type:           action,
			Subject:        subject,
			IPAddress:      &ip,
			Metadata:       datatypes.JSON(metaJSON),
		}
		activity.ID = uuid.New()

		go cfg.DB.Create(&activity)

		return err
	}
}

func mapAction(method string) string {
	switch method {
	case "POST":
		return "created"
	case "PATCH", "PUT":
		return "updated"
	case "DELETE":
		return "deleted"
	default:
		return "modified"
	}
}

func parseEntity(path string) (string, string) {
	path = strings.TrimPrefix(path, "/api/v1/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return "", ""
	}

	entityType := singularize(parts[0])

	var entityID string
	if len(parts) >= 2 && isUUID(parts[1]) {
		entityID = parts[1]
	}

	return entityType, entityID
}

func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	if strings.HasSuffix(s, "ses") {
		return strings.TrimSuffix(s, "es")
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func entityIDOrNil(s string) uuid.UUID {
	if id, err := uuid.Parse(s); err == nil {
		return id
	}
	return uuid.New()
}
