package repository

import (
	"context"
	"errors"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/routing"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ domain.QueueRepository = (*GORMQueueRepository)(nil)

type GORMQueueRepository struct {
	db *gorm.DB
}

func NewGORMQueueRepo(db *gorm.DB) *GORMQueueRepository {
	return &GORMQueueRepository{db: db}
}

// FindQueueIDsByChannel returns the queue IDs linked to the given channel via the
// channel_connection_queues join table. The join table uses snake_case columns
// (channel_connection_id, queue_id) created by GORM's many2many default naming.
func (r *GORMQueueRepository) FindQueueIDsByChannel(ctx context.Context, channelID int, tenantID uuid.UUID) ([]int, error) {
	var ids []int
	err := r.db.WithContext(ctx).
		Table("channel_connection_queues AS wq").
		Joins(`JOIN "Queues" q ON q.id = wq.queue_id`).
		Where(`wq.channel_connection_id = ? AND q."tenantId" = ?`, channelID, tenantID).
		Pluck("wq.queue_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ResolveNewTicketQueue delega o ponto único (routing) para quem só tem o repo.
func (r *GORMQueueRepository) ResolveNewTicketQueue(ctx context.Context, tenantID uuid.UUID, connectionID int) *int {
	return routing.ResolveNewTicketQueue(ctx, r.db, tenantID, connectionID)
}

func (r *GORMQueueRepository) FindByID(ctx context.Context, id int, tenantID uuid.UUID) (*domain.Queue, error) {
	var m models.Queue
	err := r.db.WithContext(ctx).
		Where("id = ? AND \"tenantId\" = ?", id, tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return queueModelToDomain(&m), nil
}

func (r *GORMQueueRepository) FindByIDWithAssociations(ctx context.Context, id int, tenantID uuid.UUID) (*domain.Queue, error) {
	var m models.Queue
	err := r.db.WithContext(ctx).
		Where("id = ? AND \"tenantId\" = ?", id, tenantID).
		Preload("ChannelConnections").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return queueModelWithAssocToDomain(&m), nil
}

func (r *GORMQueueRepository) FindAll(ctx context.Context, tenantID uuid.UUID) ([]domain.Queue, error) {
	var m []models.Queue
	err := r.db.WithContext(ctx).
		Where("\"tenantId\" = ?", tenantID).
		Find(&m).Error
	if err != nil {
		return nil, err
	}
	var queues []domain.Queue
	for _, q := range m {
		queues = append(queues, *queueModelToDomain(&q))
	}
	return queues, nil
}

func (r *GORMQueueRepository) FindAllWithAssociations(ctx context.Context, tenantID uuid.UUID) ([]domain.Queue, error) {
	var m []models.Queue
	err := r.db.WithContext(ctx).
		Where("\"tenantId\" = ?", tenantID).
		Preload("ChannelConnections").
		Order("COALESCE(\"parentId\", id), \"parentId\" IS NOT NULL, name ASC").
		Find(&m).Error
	if err != nil {
		return nil, err
	}
	var queues []domain.Queue
	for _, q := range m {
		queues = append(queues, *queueModelWithAssocToDomain(&q))
	}
	return queues, nil
}

func (r *GORMQueueRepository) Create(ctx context.Context, queue *domain.Queue, whatsappIDs []int) error {
	m := queueDomainToModel(queue)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		// Update domain ID after creation
		queue.ID = m.ID

		return models.SetWhatsappQueues(tx, queue.TenantID, m.ID, whatsappIDs)
	})
}

func (r *GORMQueueRepository) Update(ctx context.Context, queue *domain.Queue, fields map[string]interface{}, whatsappIDs []int) error {
	m := queueDomainToModel(queue)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(m).Updates(fields).Error; err != nil {
			return err
		}

		return models.SetWhatsappQueues(tx, queue.TenantID, m.ID, whatsappIDs)
	})
}

func (r *GORMQueueRepository) Save(ctx context.Context, queue *domain.Queue) error {
	m := queueDomainToModel(queue)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GORMQueueRepository) Delete(ctx context.Context, id int, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND \"tenantId\" = ?", id, tenantID).
		Delete(&models.Queue{}).Error
}

// --- Model <-> Domain mappers ---

func queueModelToDomain(m *models.Queue) *domain.Queue {
	return &domain.Queue{
		ID:                   m.ID,
		Name:                 m.Name,
		Color:                m.Color,
		GreetingMessage:      m.GreetingMessage,
		DistributionStrategy: m.DistributionStrategy,
		PrioritizeWallet:     m.PrioritizeWallet,
		ParentID:             m.ParentID,
		TenantID:             m.TenantID,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

func queueModelWithAssocToDomain(m *models.Queue) *domain.Queue {
	d := queueModelToDomain(m)
	// Map associated Whatsapp sessions to domain ChannelSession references
	for _, c := range m.ChannelConnections {
		if c.Kind != models.KindWhatsApp {
			continue
		}
		d.Whatsapps = append(d.Whatsapps, domain.QueueWhatsapp{
			ID:     c.ID,
			Name:   c.Name,
			Status: c.Status,
			Number: c.Identifier,
		})
	}
	return d
}

func queueDomainToModel(d *domain.Queue) *models.Queue {
	return &models.Queue{
		ID:                   d.ID,
		Name:                 d.Name,
		Color:                d.Color,
		GreetingMessage:      d.GreetingMessage,
		DistributionStrategy: d.DistributionStrategy,
		PrioritizeWallet:     d.PrioritizeWallet,
		ParentID:             d.ParentID,
		TenantID:             d.TenantID,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
}
