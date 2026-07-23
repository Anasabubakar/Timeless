package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type OutreachRepository struct {
	db *gorm.DB
}

func NewOutreachRepository(db *gorm.DB) *OutreachRepository {
	return &OutreachRepository{db: db}
}

func (r *OutreachRepository) ListSequences(ctx context.Context, orgID uuid.UUID, status string) ([]models.OutreachSequence, error) {
	var sequences []models.OutreachSequence
	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Preload("Steps").Order("created_at DESC").Find(&sequences).Error
	return sequences, err
}

func (r *OutreachRepository) GetSequence(ctx context.Context, orgID, id uuid.UUID) (*models.OutreachSequence, error) {
	var seq models.OutreachSequence
	err := r.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Where("organization_id = ? AND id = ?", orgID, id).
		First(&seq).Error
	if err != nil {
		return nil, err
	}
	return &seq, nil
}

func (r *OutreachRepository) CreateSequence(ctx context.Context, seq *models.OutreachSequence) error {
	return r.db.WithContext(ctx).Create(seq).Error
}

func (r *OutreachRepository) UpdateSequence(ctx context.Context, seq *models.OutreachSequence) error {
	return r.db.WithContext(ctx).Save(seq).Error
}

func (r *OutreachRepository) DeleteSequence(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.OutreachSequence{}).Error
}

func (r *OutreachRepository) ListEnrollments(ctx context.Context, orgID uuid.UUID, sequenceID *uuid.UUID, status string) ([]models.Enrollment, int64, error) {
	var enrollments []models.Enrollment
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if sequenceID != nil {
		q = q.Where("sequence_id = ?", *sequenceID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	q.Model(&models.Enrollment{}).Count(&total)
	err := q.Preload("Contact").Order("created_at DESC").Find(&enrollments).Error
	return enrollments, total, err
}

func (r *OutreachRepository) CreateEnrollment(ctx context.Context, enrollment *models.Enrollment) error {
	return r.db.WithContext(ctx).Create(enrollment).Error
}

func (r *OutreachRepository) GetEnrollmentStats(ctx context.Context, orgID, sequenceID uuid.UUID) (enrolled, opened, replied int64, err error) {
	r.db.WithContext(ctx).Model(&models.Enrollment{}).
		Where("organization_id = ? AND sequence_id = ?", orgID, sequenceID).
		Count(&enrolled)
	return enrolled, 0, 0, nil
}
