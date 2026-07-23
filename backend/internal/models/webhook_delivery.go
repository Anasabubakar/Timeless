package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type WebhookDelivery struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	WebhookID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"webhook_id"`
	Event          string         `gorm:"size:100;not null" json:"event"`
	Payload        datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	URL            string         `gorm:"type:text;not null" json:"url"`
	Status         string         `gorm:"size:20;not null;default:pending" json:"status"`
	Attempts       int            `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts    int            `gorm:"not null;default:5" json:"max_attempts"`
	ResponseCode   *int           `json:"response_code,omitempty"`
	ResponseBody   *string        `gorm:"type:text" json:"response_body,omitempty"`
	Error          *string        `gorm:"type:text" json:"error,omitempty"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`

	Webhook Webhook `gorm:"foreignKey:WebhookID" json:"-"`
}

func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}
