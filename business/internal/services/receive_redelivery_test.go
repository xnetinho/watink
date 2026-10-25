package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// O engine entrega eventos ao menos uma vez (saída durável com reenvio): a mesma message.received pode chegar duas
// vezes. A segunda não pode gravar de novo nem inflar o contador de não lidas do ticket.
func TestReceiveMessage_ReentregaDoMesmoEventoNaoDuplicaEfeitos(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Empresa"}).Error)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)

	uc := usecases.NewReceiveMessageUseCase(application.NewInMemoryEventBus(), repository.NewGORMMessageRepo(db),
		repository.NewGORMContactRepo(db), repository.NewGORMTicketRepo(db), nil, nil, nil)
	in := usecases.ReceiveMessageInput{
		ID: "3EB0REENTREGA", From: "5511999990000@s.whatsapp.net", Body: "oi", Type: "chat", PushName: "Cliente",
		Timestamp: time.Now().Unix(), SessionID: wa.ID, TenantID: tenantID,
	}

	first, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	second, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)

	var n int64
	require.NoError(t, db.Model(&models.Message{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	assert.EqualValues(t, 1, n, "a reentrega não pode gravar a mensagem de novo")

	var tk models.Ticket
	require.NoError(t, db.First(&tk, first.Ticket.ID).Error)
	assert.Equal(t, first.Ticket.ID, second.Ticket.ID)
	assert.Equal(t, 1, tk.UnreadMessages, "o contador de não lidas conta a mensagem uma única vez")
}

// Duas entregas do mesmo evento ao mesmo tempo (dois consumidores, ou reenvio da saída durável durante o tratamento):
// só uma pode contar.
func TestReceiveMessage_EntregasSimultaneasContamUmaVez(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Empresa"}).Error)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)

	uc := usecases.NewReceiveMessageUseCase(application.NewInMemoryEventBus(), repository.NewGORMMessageRepo(db),
		repository.NewGORMContactRepo(db), repository.NewGORMTicketRepo(db), nil, nil, nil)
	in := usecases.ReceiveMessageInput{
		ID: "3EB0SIMULTANEA", From: "5511988880000@s.whatsapp.net", Body: "oi", Type: "chat", PushName: "Cliente",
		Timestamp: time.Now().Unix(), SessionID: wa.ID, TenantID: tenantID,
	}
	// A primeira cria contato e ticket; as demais correm só pela mensagem.
	first, err := uc.Execute(context.Background(), usecases.ReceiveMessageInput{
		ID: "3EB0PRIMEIRA", From: in.From, Body: "antes", Type: "chat", PushName: "Cliente", Timestamp: in.Timestamp, SessionID: wa.ID, TenantID: tenantID,
	})
	require.NoError(t, err)

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, _ = uc.Execute(context.Background(), in)
		}()
	}
	wg.Wait()

	var n int64
	require.NoError(t, db.Model(&models.Message{}).Where(`"tenantId" = ? AND id = ?`, tenantID, "3EB0SIMULTANEA").Count(&n).Error)
	assert.EqualValues(t, 1, n)
	var tk models.Ticket
	require.NoError(t, db.First(&tk, first.Ticket.ID).Error)
	assert.Equal(t, 2, tk.UnreadMessages, "1 da primeira mensagem + 1 da simultânea, nunca %d", workers+1)
}
