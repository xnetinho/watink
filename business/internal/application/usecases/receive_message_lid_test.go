package usecases

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newLIDReceiveUC(db *gorm.DB) *ReceiveMessageUseCase {
	return NewReceiveMessageUseCase(
		&mockEventBus{},
		repository.NewGORMMessageRepo(db),
		repository.NewGORMContactRepo(db),
		repository.NewGORMTicketRepo(db),
		nil, nil, nil,
	)
}

func countRows(t *testing.T, db *gorm.DB, model interface{}, tenantID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(model).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	return n
}

// Cenário reportado: atendente inicia a conversa pelo contato da agenda
// (cadastrado pelo número 558382341576); a pessoa responde e o WhatsApp entrega
// o remetente como 163423740493865@lid.
func TestReceiveMessage_LIDReplyJoinsTheAgendaContactAndItsTicket(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	wa := models.Whatsapp{Name: "Conexão", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)

	agenda := models.Contact{Name: "Fulano da Agenda", Number: "558382341576", TenantID: tenant}
	require.NoError(t, db.Create(&agenda).Error)
	existing := models.Ticket{Status: "open", ContactID: agenda.ID, WhatsappID: wa.ID, TenantID: tenant}
	require.NoError(t, db.Create(&existing).Error)

	uc := newLIDReceiveUC(db)
	res, err := uc.Execute(context.Background(), ReceiveMessageInput{
		ID:        "WAMID-1",
		From:      "163423740493865@lid",
		Body:      "Oi, recebi sua mensagem",
		Type:      "chat",
		PushName:  "Fulano",
		IsLID:     true,
		ChatPN:    "558382341576@s.whatsapp.net",
		SessionID: wa.ID,
		TenantID:  tenant,
	})
	require.NoError(t, err)

	assert.Equal(t, agenda.ID, res.Contact.ID, "a resposta cai no contato da agenda")
	assert.Equal(t, existing.ID, res.Ticket.ID, "e no ticket que o atendente já tinha aberto")
	assert.Equal(t, int64(1), countRows(t, db, &models.Contact{}, tenant), "nenhum contato duplicado")
	assert.Equal(t, int64(1), countRows(t, db, &models.Ticket{}, tenant), "nenhum ticket duplicado")

	var stored models.Contact
	require.NoError(t, db.First(&stored, agenda.ID).Error)
	require.NotNil(t, stored.Lid)
	assert.Equal(t, "163423740493865@lid", *stored.Lid, "o LID fica gravado para as próximas mensagens")
	assert.Equal(t, "558382341576", stored.Number)
}

// Regressão do comportamento anterior: sem telefone resolvido, o contato por LID
// continua sendo criado (é o único jeito de atender quem não expõe número).
func TestReceiveMessage_LIDWithoutResolvedPhoneStillCreatesLIDContact(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	wa := models.Whatsapp{Name: "Conexão", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)

	res, err := newLIDReceiveUC(db).Execute(context.Background(), ReceiveMessageInput{
		ID: "WAMID-2", From: "999888777@lid", Body: "olá", Type: "chat", PushName: "Anônimo",
		IsLID: true, SessionID: wa.ID, TenantID: tenant,
	})
	require.NoError(t, err)
	require.NotNil(t, res.Contact.Lid)
	assert.Equal(t, "999888777@lid", *res.Contact.Lid)
	assert.Equal(t, int64(1), countRows(t, db, &models.Contact{}, tenant))
}

func TestReceiveMessage_PlainPhoneMessageIsUnaffected(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	wa := models.Whatsapp{Name: "Conexão", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)

	uc := newLIDReceiveUC(db)
	in := ReceiveMessageInput{ID: "A", From: "5511988887777@s.whatsapp.net", Body: "oi", Type: "chat", SessionID: wa.ID, TenantID: tenant}
	first, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	in.ID = "B"
	second, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, first.Contact.ID, second.Contact.ID)
	assert.Equal(t, "5511988887777", first.Contact.Number)
	var _ = domain.Contact{}
}
