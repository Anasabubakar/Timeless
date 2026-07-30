package middleware

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

// LogSecurityEvent records a security-relevant event as an (immutable,
// per an earlier phase's DB trigger) Activity row, async so a slow or
// failed write never adds latency to — or fails — the request that
// triggered it. This is the shared primitive behind RBAC's
// permission_denied events, the rate limiter's rate_limit_violation
// events, and the auth-event logging (login/logout/MFA/password
// changes) added in this phase — one place instead of every caller
// hand-rolling its own models.Activity construction.
//
// db may be nil (some callers, like AuthService, aren't always wired
// with one in every test/embedding context) — a nil db is a silent
// no-op rather than a panic, since a missing audit trail is a lesser
// failure than crashing the request that needed auditing in the first
// place.
func LogSecurityEvent(db *gorm.DB, orgID uuid.UUID, userID *uuid.UUID, entityType, eventType, subject, ip string, metadata map[string]string) {
	if db == nil {
		return
	}

	metaJSON, _ := json.Marshal(metadata)

	activity := models.Activity{
		OrganizationID: orgID,
		UserID:         userID,
		EntityType:     entityType,
		Type:           eventType,
		Subject:        subject,
		Metadata:       datatypes.JSON(metaJSON),
	}
	activity.ID = uuid.New()
	if ip != "" {
		activity.IPAddress = &ip
	}

	go db.Create(&activity)
}
