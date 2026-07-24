package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type CompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

func (r *CompanyRepository) Create(ctx context.Context, company *models.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *CompanyRepository) FindByID(ctx context.Context, orgID, id uuid.UUID) (*models.Company, error) {
	var company models.Company
	err := r.db.WithContext(ctx).
		Preload("Industry").Preload("DecisionMakers").
		First(&company, "id = ? AND organization_id = ?", id, orgID).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *CompanyRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Company, int64, error) {
	var companies []models.Company
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if search != "" {
		q = q.Where("name ILIKE ? OR domain ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	q.Model(&models.Company{}).Count(&total)
	err := q.Preload("Industry").
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&companies).Error
	return companies, total, err
}

func (r *CompanyRepository) Update(ctx context.Context, company *models.Company) error {
	return r.db.WithContext(ctx).Save(company).Error
}

func (r *CompanyRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Delete(&models.Company{}, "id = ?", id).Error
}
