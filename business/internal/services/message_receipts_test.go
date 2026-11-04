package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type receiptEnv struct {
	db     *gorm.DB
	tenant uuid.UUID
	el     *services.EventListener
}

func newReceiptEnv(t *testing.T) *receiptEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "Cliente", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)
	require.NoError(t, db.Create(&models.Message{ID: "OUT1", Body: "oi", FromMe: true, TicketID: tk.ID, TenantID: tenant, Ack: 1}).Error)
	require.NoError(t, db.Create(&models.Message{ID: "IN1", Body: "olá", FromMe: false, TicketID: tk.ID, TenantID: tenant}).Error)
	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(db), nil, repository.NewGORMTicketRepo(db), nil, &voBroadcast{}, db, nil, nil, nil, nil)
	return &receiptEnv{db: db, tenant: tenant, el: el}
}

func (e *receiptEnv) ack(t *testing.T, id string, ack int, at, by string) {
	t.Helper()
	p, _ := json.Marshal(map[string]interface{}{"messageId": id, "ack": ack, "at": at, "by": by})
	require.NoError(t, e.el.HandleMessageAckEvent(context.Background(), p, e.tenant))
}

func (e *receiptEnv) receipts(t *testing.T, id string) []services.ReceiptRow {
	t.Helper()
	r, err := services.MessageReceipts(context.Background(), e.db, e.tenant, id)
	require.NoError(t, err)
	return r
}

func TestMessageReceipts_ConversaIndividual(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "OUT1", 2, "2026-10-30T14:00:00Z", "")
	e.ack(t, "OUT1", 3, "2026-10-30T14:05:00Z", "")

	rows := e.receipts(t, "OUT1")
	require.Len(t, rows, 1, "conversa individual = uma linha só")
	assert.Equal(t, "2026-10-30T14:00:00Z", *rows[0].Delivered)
	assert.Equal(t, "2026-10-30T14:05:00Z", *rows[0].Read)
	assert.Nil(t, rows[0].Played)

	var m models.Message
	require.NoError(t, e.db.Where("id = ?", "OUT1").Take(&m).Error)
	assert.Equal(t, 3, m.Ack, "o ack da mensagem continua sendo o estágio mais alto")
}

// O recibo repetido (o engine entrega ao menos uma vez) mantém o PRIMEIRO horário e não duplica.
func TestMessageReceipts_Idempotente(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "OUT1", 3, "2026-10-30T14:05:00Z", "")
	e.ack(t, "OUT1", 3, "2026-10-30T18:00:00Z", "")
	rows := e.receipts(t, "OUT1")
	require.Len(t, rows, 1)
	assert.Equal(t, "2026-10-30T14:05:00Z", *rows[0].Read)
	var n int64
	e.db.Model(&models.MessageReceipt{}).Where(`"messageId" = ?`, "OUT1").Count(&n)
	assert.EqualValues(t, 1, n)
}

// Entregue que chega DEPOIS do lido (fora de ordem): o ack não desce, mas o horário de entrega é um fato e é gravado.
func TestMessageReceipts_ForaDeOrdem(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "OUT1", 3, "2026-10-30T14:05:00Z", "")
	e.ack(t, "OUT1", 2, "2026-10-30T14:00:00Z", "")
	rows := e.receipts(t, "OUT1")
	require.Len(t, rows, 1)
	assert.Equal(t, "2026-10-30T14:00:00Z", *rows[0].Delivered)
	assert.Equal(t, "2026-10-30T14:05:00Z", *rows[0].Read)
	var m models.Message
	require.NoError(t, e.db.Where("id = ?", "OUT1").Take(&m).Error)
	assert.Equal(t, 3, m.Ack, "ack não desce")
}

// Grupo: cada participante manda o seu recibo; o nome sai do contato quando o JID casa.
func TestMessageReceipts_GrupoPorParticipante(t *testing.T) {
	e := newReceiptEnv(t)
	require.NoError(t, e.db.Create(&models.Contact{Name: "Maria", Number: "5511988887777", TenantID: e.tenant}).Error)
	e.ack(t, "OUT1", 2, "2026-10-30T14:00:00Z", "5511988887777@s.whatsapp.net")
	e.ack(t, "OUT1", 3, "2026-10-30T14:10:00Z", "5511988887777@s.whatsapp.net")
	e.ack(t, "OUT1", 2, "2026-10-30T14:01:00Z", "5511977776666@s.whatsapp.net")

	rows := e.receipts(t, "OUT1")
	require.Len(t, rows, 2)
	byJID := map[string]services.ReceiptRow{}
	for _, r := range rows {
		byJID[r.SenderJID] = r
	}
	maria := byJID["5511988887777@s.whatsapp.net"]
	assert.Equal(t, "Maria", maria.Name)
	assert.NotNil(t, maria.Read)
	outro := byJID["5511977776666@s.whatsapp.net"]
	assert.Empty(t, outro.Name, "sem contato, sem nome (a tela mostra o número)")
	assert.Nil(t, outro.Read)
}

// Recibo só vale para mensagem NOSSA: mensagem recebida não gera dados de entrega.
func TestMessageReceipts_SoMensagemNossa(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "IN1", 3, "2026-10-30T14:00:00Z", "")
	assert.Empty(t, e.receipts(t, "IN1"))
}

// Ack 1 (enviado ao servidor) e 5 (erro) não são recibos do contato; horário ausente ou inválido usa o da chegada.
func TestMessageReceipts_AckQueNaoEhRecibo_EHorarioInvalido(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "OUT1", 1, "2026-10-30T14:00:00Z", "")
	e.ack(t, "OUT1", 5, "2026-10-30T14:00:00Z", "")
	assert.Empty(t, e.receipts(t, "OUT1"))

	before := time.Now().Add(-time.Second)
	e.ack(t, "OUT1", 2, "lixo", "")
	rows := e.receipts(t, "OUT1")
	require.Len(t, rows, 1)
	got, err := time.Parse("2006-01-02T15:04:05Z07:00", *rows[0].Delivered)
	require.NoError(t, err)
	assert.False(t, got.Before(before.UTC().Truncate(time.Second)))
}

func TestMessageReceipts_OutraEmpresaNaoVe(t *testing.T) {
	e := newReceiptEnv(t)
	e.ack(t, "OUT1", 3, "2026-10-30T14:05:00Z", "")
	other, err := services.MessageReceipts(context.Background(), e.db, uuid.New(), "OUT1")
	require.NoError(t, err)
	assert.Empty(t, other)
}
