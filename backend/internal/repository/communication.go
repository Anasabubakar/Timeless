package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type CommunicationRepository struct {
	db *gorm.DB
}

func NewCommunicationRepository(db *gorm.DB) *CommunicationRepository {
	return &CommunicationRepository{db: db}
}

func (r *CommunicationRepository) List(orgID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]models.Communication, int64, error) {
	var comms []models.Communication
	var total int64

	q := r.db.Where("organization_id = ?", orgID)

	if status, ok := filters["status"]; ok {
		q = q.Where("status = ?", status)
	}
	if commType, ok := filters["type"]; ok {
		q = q.Where("type = ?", commType)
	}
	if direction, ok := filters["direction"]; ok {
		q = q.Where("direction = ?", direction)
	}
	if sponsorID, ok := filters["sponsor_id"]; ok {
		q = q.Where("sponsor_id = ?", sponsorID)
	}
	if contactID, ok := filters["contact_id"]; ok {
		q = q.Where("contact_id = ?", contactID)
	}

	q.Model(&models.Communication{}).Count(&total)

	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&comms).Error
	return comms, total, err
}

func (r *CommunicationRepository) GetByID(orgID, id uuid.UUID) (*models.Communication, error) {
	var comm models.Communication
	err := r.db.Where("organization_id = ? AND id = ?", orgID, id).First(&comm).Error
	if err != nil {
		return nil, err
	}
	return &comm, nil
}

func (r *CommunicationRepository) Create(comm *models.Communication) error {
	return r.db.Create(comm).Error
}

func (r *CommunicationRepository) Update(comm *models.Communication) error {
	return r.db.Save(comm).Error
}

func (r *CommunicationRepository) Delete(orgID, id uuid.UUID) error {
	return r.db.Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Communication{}).Error
}

func (r *CommunicationRepository) GetStats(orgID uuid.UUID) (map[string]int64, error) {
	stats := make(map[string]int64)

	var sent, opened, clicked, replied, bounced int64
	base := r.db.Model(&models.Communication{}).Where("organization_id = ?", orgID)

	base.Where("status = ?", "sent").Count(&sent)
	base.Where("opened_at IS NOT NULL").Count(&opened)
	base.Where("clicked_at IS NOT NULL").Count(&clicked)
	base.Where("replied_at IS NOT NULL").Count(&replied)
	base.Where("bounced_at IS NOT NULL").Count(&bounced)

	stats["sent"] = sent
	stats["opened"] = opened
	stats["clicked"] = clicked
	stats["replied"] = replied
	stats["bounced"] = bounced

	return stats, nil
}
