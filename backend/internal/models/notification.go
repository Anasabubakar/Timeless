package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type NotificationType string

const (
	NotifAgentComplete   NotificationType = "agent.complete"
	NotifPipelineMove    NotificationType = "pipeline.move"
	NotifOutreachReply   NotificationType = "outreach.reply"
	NotifMention         NotificationType = "mention"
	NotifDealWon         NotificationType = "deal.won"
	NotifDealLost        NotificationType = "deal.lost"
	NotifTaskAssigned    NotificationType = "task.assigned"
	NotifSequenceStep    NotificationType = "sequence.step"
	NotifWebhookFailed   NotificationType = "webhook.failed"
	NotifTeamInvite      NotificationType = "team.invite"
	NotifSystemAlert     NotificationType = "system.alert"
)

type Notification struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	Type           NotificationType `gorm:"size:50;not null;index" json:"type"`
	Title          string           `gorm:"size:255;not null" json:"title"`
	Body           string           `gorm:"type:text" json:"body"`
	ActionURL      string           `gorm:"size:500" json:"action_url,omitempty"`
	EntityType     string           `gorm:"size:50" json:"entity_type,omitempty"`
	EntityID       *uuid.UUID       `gorm:"type:uuid" json:"entity_id,omitempty"`
	Metadata       datatypes.JSON   `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	Read           bool             `gorm:"default:false;index" json:"read"`
	ReadAt         *time.Time       `json:"read_at,omitempty"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

type NotificationPreference struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null" json:"organization_id"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	Type           NotificationType `gorm:"size:50;not null" json:"type"`
	InApp          bool             `gorm:"default:true" json:"in_app"`
	Email          bool             `gorm:"default:false" json:"email"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}
