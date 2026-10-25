package application_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurableOutboxEventBus_RecoversUnprocessedEventsAfterCrash(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	// 1. Simula processo que caiu após gravar no banco mas antes do handler executar
	initialRow := models.OutboxEvent{
		TenantID:  tenantID,
		EventName: "MessageReceived",
		Payload:   `{"messageId":"3EB0CRASHED1","ticketId":10,"tenantId":"` + tenantID.String() + `"}`,
		CreatedAt: time.Now(),
	}
	require.NoError(t, db.Create(&initialRow).Error)

	// 2. Novo processo sobe, registra subscriber e recupera pendentes
	bus := application.NewDurableOutboxEventBus(db)
	var handledCount atomic.Int32
	var lastMessageID string

	require.NoError(t, bus.Subscribe("MessageReceived", func(ctx context.Context, event domain.DomainEvent) error {
		handledCount.Add(1)
		if mr, ok := event.(domain.MessageReceivedEvent); ok {
			lastMessageID = mr.MessageID
		}
		return nil
	}))

	n, err := bus.ProcessPending(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "deve processar o evento que sobreviveu à queda")

	assert.Equal(t, int32(1), handledCount.Load())
	assert.Equal(t, "3EB0CRASHED1", lastMessageID)

	// 3. Verifica no banco que foi marcado como processado
	var updated models.OutboxEvent
	require.NoError(t, db.First(&updated, initialRow.ID).Error)
	assert.NotNil(t, updated.ProcessedAt, "o evento recuperado deve ser marcado como processado")

	// 4. Segundo ProcessPending não deve reprocessar
	n2, err := bus.ProcessPending(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "não deve reprocessar eventos já marcados")
}

func TestDurableOutboxEventBus_PublishWritesToDatabase(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	bus := application.NewDurableOutboxEventBus(db)
	var handled atomic.Bool
	require.NoError(t, bus.Subscribe("TicketAssigned", func(ctx context.Context, event domain.DomainEvent) error {
		handled.Store(true)
		return nil
	}))

	event := domain.NewTicketAssignedEvent(55, 99, tenantID)
	require.NoError(t, bus.Publish(context.Background(), event))

	// Aguarda processamento assíncrono
	time.Sleep(100 * time.Millisecond)
	assert.True(t, handled.Load())

	var saved models.OutboxEvent
	require.NoError(t, db.Where(`"tenantId" = ? AND "eventName" = ?`, tenantID, "TicketAssigned").First(&saved).Error)
	assert.Equal(t, "TicketAssigned", saved.EventName)
}
