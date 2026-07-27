package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) List(ctx context.Context, orgID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&projects).Error
	return projects, err
}

func (r *ProjectRepository) GetByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Project, error) {
	var project models.Project
	err := r.db.WithContext(ctx).Where("organization_id = ? AND name = ?", orgID, name).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}
