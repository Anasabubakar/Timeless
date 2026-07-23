package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").
		First(&user, "organization_id = ? AND email = ?", orgID, email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	q.Model(&models.User{}).Count(&total)
	err := q.Preload("Roles").Limit(limit).Offset(offset).Order("created_at DESC").Find(&users).Error
	return users, total, err
}
