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
	"github.com/timeless/backend/internal/realtime"
)

type AuditConfig struct {
	DB *gorm.DB
	// Hub broadcasts an EventActivity to every other connected member of
	// the org for every audited mutation, powering the realtime
	// "so-and-so updated Sponsors" notification banner. Nil is valid
	// (e.g. in tests) — broadcasting is skipped, only the DB write
	// happens.
	Hub *realtime.Hub
}

func AuditLog(cfg AuditConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()

		status := c.Response().StatusCode()
		method := c.Method()

		if !shouldAudit(method, status) {
			return err
		}

		orgID := GetOrgID(c)
		userID := GetUserID(c)
		if orgID == uuid.Nil {
			return err
		}

		path := c.Path()
		action := mapAction(method, status >= 500)
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

		// Only broadcast successful, non-read mutations — a failed write
		// or a 4xx never actually changed anything another member's UI
		// needs to react to (shouldAudit already let 5xx through so it
		// still lands in the DB audit trail above, but nobody else's
		// screen went stale because of it).
		if cfg.Hub != nil && status < 300 {
			actorEmail, _ := c.Locals("email").(string)
			cfg.Hub.Publish(&realtime.Event{
				Type:  realtime.EventActivity,
				OrgID: orgID.String(),
				Payload: map[string]interface{}{
					"actor_email": actorEmail,
					"action":      action,
					"entity_type": entityType,
					"entity_id":   entityID,
					"subject":     subject,
				},
			})
		}

		return err
	}
}

// shouldAudit decides whether a request is a candidate for the general
// audit trail: mutating (not GET/OPTIONS/HEAD), and either succeeded or
// hit a server error. 401/403/429 are deliberately excluded — auth
// failure never reaches this middleware at all, and RBAC denial /
// rate-limit violations already log themselves via LogSecurityEvent, so
// re-logging them here would just be a noisier duplicate.
func shouldAudit(method string, status int) bool {
	if method == "GET" || method == "OPTIONS" || method == "HEAD" {
		return false
	}
	return status < 400 || status >= 500
}

func mapAction(method string, failed bool) string {
	action := "modified"
	switch method {
	case "POST":
		action = "created"
	case "PATCH", "PUT":
		action = "updated"
	case "DELETE":
		action = "deleted"
	}
	if failed {
		return "failed_" + action
	}
	return action
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
