package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepository) CreateBatch(ctx context.Context, notifications []*models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&notifications).Error
}

func (r *NotificationRepository) List(ctx context.Context, orgID, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var count int64

	q := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", orgID, userID)

	if unreadOnly {
		q = q.Where("read = false")
	}

	if err := q.Model(&models.Notification{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	err := q.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, count, err
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, orgID, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("organization_id = ? AND user_id = ? AND read = false", orgID, userID).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, orgID, userID, notifID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("id = ? AND organization_id = ? AND user_id = ?", notifID, orgID, userID).
		Updates(map[string]interface{}{"read": true, "read_at": now}).Error
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, orgID, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("organization_id = ? AND user_id = ? AND read = false", orgID, userID).
		Updates(map[string]interface{}{"read": true, "read_at": now}).Error
}

func (r *NotificationRepository) Delete(ctx context.Context, orgID, userID, notifID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ? AND user_id = ?", notifID, orgID, userID).
		Delete(&models.Notification{}).Error
}

func (r *NotificationRepository) GetPreferences(ctx context.Context, orgID, userID uuid.UUID) ([]models.NotificationPreference, error) {
	var prefs []models.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Find(&prefs).Error
	return prefs, err
}

func (r *NotificationRepository) UpsertPreference(ctx context.Context, pref *models.NotificationPreference) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ? AND type = ?",
			pref.OrganizationID, pref.UserID, pref.Type).
		Assign(models.NotificationPreference{
			InApp: pref.InApp,
			Email: pref.Email,
		}).
		FirstOrCreate(pref).Error
}

func (r *NotificationRepository) IsTypeMuted(ctx context.Context, orgID, userID uuid.UUID, notifType models.NotificationType) (bool, error) {
	var pref models.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ? AND type = ?", orgID, userID, notifType).
		First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !pref.InApp, nil
}
