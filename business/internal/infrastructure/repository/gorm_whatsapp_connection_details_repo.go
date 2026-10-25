package repository

import (
	"context"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"gorm.io/gorm"
)

type GORMWhatsAppConnectionDetailsRepository struct{ db *gorm.DB }

func NewGORMWhatsAppConnectionDetailsRepo(db *gorm.DB) *GORMWhatsAppConnectionDetailsRepository {
	return &GORMWhatsAppConnectionDetailsRepository{db: db}
}

func (r *GORMWhatsAppConnectionDetailsRepository) FindByConnectionID(ctx context.Context, connectionID int) (*models.WhatsAppConnectionDetails, error) {
	var details models.WhatsAppConnectionDetails
	if err := r.db.WithContext(ctx).First(&details, `"connectionId" = ?`, connectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &details, nil
}

func (r *GORMWhatsAppConnectionDetailsRepository) Save(ctx context.Context, details *models.WhatsAppConnectionDetails) error {
	return r.db.WithContext(ctx).Save(details).Error
}
