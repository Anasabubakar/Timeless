package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type ContactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Contact, int64, error) {
	var contacts []models.Contact
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if search != "" {
		q = q.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	q.Model(&models.Contact{}).Count(&total)
	err := q.Preload("Company").Order("created_at DESC").Limit(limit).Offset(offset).Find(&contacts).Error
	return contacts, total, err
}

func (r *ContactRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.WithContext(ctx).Preload("Company").Where("organization_id = ? AND id = ?", orgID, id).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) Create(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

func (r *ContactRepository) Update(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Save(contact).Error
}

func (r *ContactRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Contact{}).Error
}
