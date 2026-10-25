package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/safego"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DurableOutboxEventBus implements domain.EventBus with transactional outbox persistence.
// Events are written to the database before processing so they survive process restarts.
type DurableOutboxEventBus struct {
	db          *gorm.DB
	mu          sync.RWMutex
	subscribers map[string][]domain.EventHandler
}

func NewDurableOutboxEventBus(db *gorm.DB) *DurableOutboxEventBus {
	return &DurableOutboxEventBus{
		db:          db,
		subscribers: make(map[string][]domain.EventHandler),
	}
}

// Publish serializes and persists the domain event in OutboxEvents, then dispatches to subscribers.
func (b *DurableOutboxEventBus) Publish(ctx context.Context, event domain.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("outbox: marshal event %q: %w", event.EventName(), err)
	}

	var outboxID int64
	if b.db != nil {
		row := models.OutboxEvent{
			TenantID:  event.TenantID(),
			EventName: event.EventName(),
			Payload:   string(payload),
			CreatedAt: time.Now(),
		}
		if err := b.db.WithContext(ctx).Create(&row).Error; err != nil {
			return fmt.Errorf("outbox: persist event %q: %w", event.EventName(), err)
		}
		outboxID = row.ID
	}

	b.mu.RLock()
	handlers, ok := b.subscribers[event.EventName()]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return nil
	}

	for _, handler := range handlers {
		h := handler
		safego.Go("eventbus."+event.EventName(), func() {
			if err := h(ctx, event); err != nil {
				log.Printf("outbox: handler error for event %q: %v", event.EventName(), err)
				return
			}
			if b.db != nil && outboxID > 0 {
				now := time.Now()
				_ = b.db.Session(&gorm.Session{NewDB: true}).
					Model(&models.OutboxEvent{}).
					Where(`id = ? AND "tenantId" = ?`, outboxID, event.TenantID()).
					Update("processedAt", now).Error
			}
		})
	}

	return nil
}

// Subscribe registers an event handler.
func (b *DurableOutboxEventBus) Subscribe(eventName string, handler domain.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventName] = append(b.subscribers[eventName], handler)
	return nil
}

// ProcessPending recovers and dispatches any unprocessed events (e.g. after a process crash).
func (b *DurableOutboxEventBus) ProcessPending(ctx context.Context) (int, error) {
	if b.db == nil {
		return 0, nil
	}

	var pending []models.OutboxEvent
	if err := b.db.WithContext(ctx).
		Where(`"processedAt" IS NULL`).
		Order("id ASC").
		Limit(100).
		Find(&pending).Error; err != nil {
		return 0, err
	}

	processed := 0
	for _, row := range pending {
		event, err := deserializeDomainEvent(row.EventName, row.TenantID, []byte(row.Payload))
		if err != nil {
			log.Printf("outbox: deserialize error for event %d (%s): %v", row.ID, row.EventName, err)
			continue
		}

		b.mu.RLock()
		handlers := b.subscribers[row.EventName]
		b.mu.RUnlock()

		allSucceeded := true
		for _, h := range handlers {
			if err := h(ctx, event); err != nil {
				log.Printf("outbox: recovered handler error for event %d (%s): %v", row.ID, row.EventName, err)
				allSucceeded = false
			}
		}

		if allSucceeded {
			now := time.Now()
			_ = b.db.WithContext(ctx).
				Model(&models.OutboxEvent{}).
				Where(`id = ? AND "tenantId" = ?`, row.ID, row.TenantID).
				Update("processedAt", now).Error
			processed++
		}
	}

	return processed, nil
}

type genericDomainEvent struct {
	name     string
	tenantID uuid.UUID
	payload  map[string]any
}

func (e genericDomainEvent) EventName() string   { return e.name }
func (e genericDomainEvent) TenantID() uuid.UUID { return e.tenantID }

func deserializeDomainEvent(name string, tenantID uuid.UUID, data []byte) (domain.DomainEvent, error) {
	switch name {
	case "MessageReceived":
		var e domain.MessageReceivedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "TicketAssigned":
		var e domain.TicketAssignedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "TicketStatusChanged":
		var e domain.TicketStatusChangedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		return genericDomainEvent{name: name, tenantID: tenantID, payload: m}, nil
	}
}
