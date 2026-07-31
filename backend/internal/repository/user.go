package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
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

// CountByOrg returns the total number of members in an organization —
// used by account self-deletion to tell "sole member" (deleting them
// takes the org with them) from "one of several" (deleting an Owner
// without transferring first would leave the org ownerless).
func (r *UserRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("organization_id = ?", orgID).Count(&count).Error
	return count, err
}

// Delete soft-deletes a user (gorm.Model's DeletedAt), consistent with
// every other delete path in the app (e.g. team member removal) — the
// row stays for audit/FK integrity but is excluded from normal queries.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id).Error
}
