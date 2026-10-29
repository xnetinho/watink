package calls

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Testes caóticos de chamada: UDP bloqueado, perda de relay e reinício do engine no meio da chamada (Tarefa 6.4).
// Verifica que a chamada termina com o motivo correto e libera a conexão para novas chamadas.

func TestChaotic_UDPBlocked_EndsAsMediaTimeoutAndFreesConnection(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "da_fila_A", "place")
	r.online("da_fila_A")

	callID := "CHAOS-UDP"
	require.NoError(t, r.svc.HandleIncoming(context.Background(), r.tenant, incoming(callID, r.waA.ID, "5511999990021@s.whatsapp.net", "5511999990021")))
	_, err := r.svc.Accept(context.Background(), r.tenant, r.users["da_fila_A"].ID, callID)
	require.NoError(t, err)

	// Simula UDP bloqueado: mídia nunca conecta, engine dispara término por media_timeout
	require.NoError(t, r.svc.HandleEnded(context.Background(), r.tenant, ended(callID, "media_timeout", 0)))

	l := r.log(t, callID)
	assert.Equal(t, StatusFailed, l.Status)
	assert.Equal(t, "media_timeout", l.EndReason)
	assert.NotNil(t, l.EndedAt)

	// Valida que a conexão foi liberada: uma nova chamada pode ser originada sem ErrConnectionBusy
	contact := models.Contact{Name: "Bob", Number: "5511999990022", TenantID: r.tenant}
	require.NoError(t, r.db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: r.waA.ID, TenantID: r.tenant}
	require.NoError(t, r.db.Create(&ticket).Error)

	placed, err := r.svc.Place(context.Background(), r.tenant, r.users["da_fila_A"].ID, &ticket, &contact)
	require.NoError(t, err, "conexão deve estar livre após término por UDP bloqueado")
	assert.NotNil(t, placed)
}

func TestChaotic_RelayLoss_EndsAsFailedAndFreesConnection(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "da_fila_A", "place")
	r.online("da_fila_A")

	callID := "CHAOS-RELAY"
	require.NoError(t, r.svc.HandleIncoming(context.Background(), r.tenant, incoming(callID, r.waA.ID, "5511999990023@s.whatsapp.net", "5511999990023")))
	_, err := r.svc.Accept(context.Background(), r.tenant, r.users["da_fila_A"].ID, callID)
	require.NoError(t, err)

	// Simula perda irreversível de relay no meio da conversa
	require.NoError(t, r.svc.HandleEnded(context.Background(), r.tenant, ended(callID, "failed", 12)))

	l := r.log(t, callID)
	assert.Equal(t, StatusFailed, l.Status)
	assert.Equal(t, "failed", l.EndReason)
	assert.NotNil(t, l.EndedAt)

	// Conexão liberada para nova chamada
	contact := models.Contact{Name: "Carlos", Number: "5511999990024", TenantID: r.tenant}
	require.NoError(t, r.db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: r.waA.ID, TenantID: r.tenant}
	require.NoError(t, r.db.Create(&ticket).Error)

	placed, err := r.svc.Place(context.Background(), r.tenant, r.users["da_fila_A"].ID, &ticket, &contact)
	require.NoError(t, err, "conexão deve estar livre após falha de relay")
	assert.NotNil(t, placed)
}

func TestChaotic_EngineRestartMidCall_EndsAsInterruptedAndFreesConnection(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "da_fila_A", "place")
	r.online("da_fila_A")

	callID := "CHAOS-RESTART"
	require.NoError(t, r.svc.HandleIncoming(context.Background(), r.tenant, incoming(callID, r.waA.ID, "5511999990025@s.whatsapp.net", "5511999990025")))
	_, err := r.svc.Accept(context.Background(), r.tenant, r.users["da_fila_A"].ID, callID)
	require.NoError(t, err)

	// Simula reinício do engine: publica evento call.reset daquela conexão
	resetPayload, _ := json.Marshal(map[string]interface{}{"sessionId": itoa(r.waA.ID)})
	require.NoError(t, r.svc.HandleReset(context.Background(), r.tenant, resetPayload))

	l := r.log(t, callID)
	assert.Equal(t, StatusInterrupted, l.Status)
	assert.Equal(t, "interrupted", l.EndReason)
	assert.NotNil(t, l.EndedAt)

	// Conexão liberada para nova chamada
	contact := models.Contact{Name: "Daniel", Number: "5511999990026", TenantID: r.tenant}
	require.NoError(t, r.db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: r.waA.ID, TenantID: r.tenant}
	require.NoError(t, r.db.Create(&ticket).Error)

	placed, err := r.svc.Place(context.Background(), r.tenant, r.users["da_fila_A"].ID, &ticket, &contact)
	require.NoError(t, err, "conexão deve estar livre após recuperação de reinício do engine")
	assert.NotNil(t, placed)
}
