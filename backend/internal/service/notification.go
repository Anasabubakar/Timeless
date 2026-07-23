package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/realtime"
	"github.com/sponsoros/backend/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepository
	hub  *realtime.Hub
}

func NewNotificationService(repo *repository.NotificationRepository, hub *realtime.Hub) *NotificationService {
	return &NotificationService{repo: repo, hub: hub}
}

type SendNotificationInput struct {
	OrgID      uuid.UUID
	UserID     uuid.UUID
	Type       models.NotificationType
	Title      string
	Body       string
	ActionURL  string
	EntityType string
	EntityID   *uuid.UUID
	Metadata   map[string]interface{}
}

func (s *NotificationService) Send(ctx context.Context, input *SendNotificationInput) (*models.Notification, error) {
	muted, _ := s.repo.IsTypeMuted(ctx, input.OrgID, input.UserID, input.Type)
	if muted {
		return nil, nil
	}

	n := &models.Notification{
		OrganizationID: input.OrgID,
		UserID:         input.UserID,
		Type:           input.Type,
		Title:          input.Title,
		Body:           input.Body,
		ActionURL:      input.ActionURL,
		EntityType:     input.EntityType,
		EntityID:       input.EntityID,
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	if s.hub != nil {
		s.hub.Publish(&realtime.Event{
			Type:  realtime.EventNotification,
			OrgID: input.OrgID.String(),
			Payload: map[string]interface{}{
				"id":      n.ID.String(),
				"type":    string(n.Type),
				"title":   n.Title,
				"body":    n.Body,
				"user_id": input.UserID.String(),
			},
		})
	}

	return n, nil
}

func (s *NotificationService) SendToOrg(ctx context.Context, orgID uuid.UUID, userIDs []uuid.UUID, notifType models.NotificationType, title, body, actionURL string) error {
	var notifications []*models.Notification
	for _, uid := range userIDs {
		muted, _ := s.repo.IsTypeMuted(ctx, orgID, uid, notifType)
		if muted {
			continue
		}
		notifications = append(notifications, &models.Notification{
			OrganizationID: orgID,
			UserID:         uid,
			Type:           notifType,
			Title:          title,
			Body:           body,
			ActionURL:      actionURL,
		})
	}

	if len(notifications) == 0 {
		return nil
	}

	if err := s.repo.CreateBatch(ctx, notifications); err != nil {
		return err
	}

	if s.hub != nil {
		s.hub.Publish(&realtime.Event{
			Type:  realtime.EventNotification,
			OrgID: orgID.String(),
			Payload: map[string]interface{}{
				"type":  string(notifType),
				"title": title,
				"body":  body,
			},
		})
	}

	return nil
}

func (s *NotificationService) List(ctx context.Context, orgID, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]models.Notification, int64, error) {
	return s.repo.List(ctx, orgID, userID, unreadOnly, limit, offset)
}

func (s *NotificationService) UnreadCount(ctx context.Context, orgID, userID uuid.UUID) (int64, error) {
	return s.repo.UnreadCount(ctx, orgID, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, orgID, userID, notifID uuid.UUID) error {
	return s.repo.MarkRead(ctx, orgID, userID, notifID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, orgID, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, orgID, userID)
}

func (s *NotificationService) Delete(ctx context.Context, orgID, userID, notifID uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, userID, notifID)
}

func (s *NotificationService) GetPreferences(ctx context.Context, orgID, userID uuid.UUID) ([]models.NotificationPreference, error) {
	return s.repo.GetPreferences(ctx, orgID, userID)
}

func (s *NotificationService) UpdatePreference(ctx context.Context, pref *models.NotificationPreference) error {
	return s.repo.UpsertPreference(ctx, pref)
}
