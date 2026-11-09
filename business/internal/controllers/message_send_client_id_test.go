package controllers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payloadCapture guarda o payload do evento appMessage da sala do ticket.
type payloadCapture struct {
	mu      sync.Mutex
	payload []interface{}
}

func (b *payloadCapture) EmitToRoom(_, _, event string, p interface{}) {
	if event != "appMessage" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.payload = append(b.payload, p)
}
func (b *payloadCapture) EmitToTenantRoom(string, string, interface{}) {}
func (b *payloadCapture) EmitToNamespace(_, _ string, _ interface{})   {}

// A tela mostra a mensagem na hora, antes do POST responder (envio otimista). O id local volta no evento e na resposta para ela trocar a cópia local pela
// real sem duplicar, e NUNCA é gravado no banco.
func TestSendMessage_DevolveOClientMessageIDNoEventoENaResposta(t *testing.T) {
	bc := &payloadCapture{}
	e := newSendEnvWithBroadcast(t, bc)
	ticket := e.newTicket(t, "tg:111")

	w := e.send(t, ticket.ID, `{"body":"oi","clientMessageId":"c-123"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		ID              string `json:"id"`
		ClientMessageID string `json:"clientMessageId"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "c-123", resp.ClientMessageID)

	require.Len(t, bc.payload, 1)
	raw, err := json.Marshal(bc.payload[0])
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"clientMessageId":"c-123"`)

	var saved models.Message
	require.NoError(t, e.db.Where(`"ticketId" = ?`, ticket.ID).First(&saved).Error)
	assert.Equal(t, resp.ID, saved.ID)
}

func TestSendMessage_ClientMessageIDGrandeDemaisERecusado(t *testing.T) {
	e := newSendEnv(t)
	ticket := e.newTicket(t, "tg:111")
	w := e.send(t, ticket.ID, `{"body":"oi","clientMessageId":"`+strings.Repeat("x", 65)+`"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
