package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ domain.QuickAnswerRepository = (*GORMQuickAnswerRepository)(nil)

type GORMQuickAnswerRepository struct {
	db *gorm.DB
}

func NewGORMQuickAnswerRepo(db *gorm.DB) *GORMQuickAnswerRepository {
	return &GORMQuickAnswerRepository{db: db}
}

func (r *GORMQuickAnswerRepository) Find(ctx context.Context, tenantID uuid.UUID, search string) ([]models.QuickAnswer, error) {
	var list []models.QuickAnswer
	query := r.db.WithContext(ctx).Where("\"tenantId\" = ?", tenantID)
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		query = query.Where(
			"LOWER(shortcut) LIKE ? OR LOWER(slug) LIKE ? OR LOWER(message) LIKE ?",
			like, like, like,
		)
	}
	err := query.Order("shortcut ASC").Find(&list).Error
	return list, err
}

func (r *GORMQuickAnswerRepository) FindByID(ctx context.Context, id int, tenantID uuid.UUID) (*models.QuickAnswer, error) {
	var qa models.QuickAnswer
	err := r.db.WithContext(ctx).
		Where("id = ? AND \"tenantId\" = ?", id, tenantID).
		First(&qa).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &qa, nil
}

func (r *GORMQuickAnswerRepository) Create(ctx context.Context, qa *models.QuickAnswer) error {
	return r.db.WithContext(ctx).Create(qa).Error
}

func (r *GORMQuickAnswerRepository) Update(ctx context.Context, qa *models.QuickAnswer) error {
	return r.db.WithContext(ctx).Save(qa).Error
}

func (r *GORMQuickAnswerRepository) Delete(ctx context.Context, id int, tenantID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND \"tenantId\" = ?", id, tenantID).
		Delete(&models.QuickAnswer{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
