package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditRefusal(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	ok := models.Message{FromMe: true, Ack: 2, CreatedAt: now.Add(-5 * time.Minute)}
	with := func(f func(*models.Message)) models.Message { m := ok; f(&m); return m }

	cases := []struct {
		name string
		m    models.Message
		code string
	}{
		{"nossa, entregue, dentro do prazo: pode", ok, ""},
		{"ack 1 (celular desligado) também pode: o servidor já aceitou", with(func(m *models.Message) { m.Ack = 1 }), ""},
		{"legenda de imagem pode", with(func(m *models.Message) { m.MediaType = "image" }), ""},
		{"exatamente nos 20 min ainda pode", with(func(m *models.Message) { m.CreatedAt = now.Add(-EditWindow) }), ""},
		{"20 min e 1 s: prazo vencido", with(func(m *models.Message) { m.CreatedAt = now.Add(-EditWindow - time.Second) }), "edit_window_expired"},
		{"recebida do contato: não é nossa", with(func(m *models.Message) { m.FromMe = false }), "message_not_editable"},
		{"apagada", with(func(m *models.Message) { m.IsDeleted = true }), "message_not_editable"},
		{"áudio não tem texto a editar", with(func(m *models.Message) { m.MediaType = "audio" }), "message_not_editable"},
		{"enquete", with(func(m *models.Message) { m.MediaType = "poll" }), "message_not_editable"},
		{"ack 0: ainda não saiu do engine", with(func(m *models.Message) { m.Ack = 0 }), "message_not_sent"},
		{"ack 5: falhou (o caminho é reenviar, não editar)", with(func(m *models.Message) { m.Ack = 5 }), "message_not_sent"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _ := editRefusal(c.m, now)
			assert.Equal(t, c.code, code)
		})
	}
}

func TestEditMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	bc := &captureBroadcaster{}
	mc := NewMessageController(pub, bc, nil)

	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	require.NoError(t, db.Create(&ticket).Error)
	mk := func(id string, mut func(*models.Message)) {
		m := models.Message{ID: id, Body: "texto antigo", TicketID: ticket.ID, FromMe: true, Ack: 2, ContactID: &contact.ID, TenantID: tenantID, CreatedAt: time.Now().Add(-2 * time.Minute)}
		if mut != nil {
			mut(&m)
		}
		require.NoError(t, db.Create(&m).Error)
	}
	mk("OK", nil)
	mk("VELHA", func(m *models.Message) { m.CreatedAt = time.Now().Add(-30 * time.Minute) })
	mk("RECEBIDA", func(m *models.Message) { m.FromMe = false })
	mk("IMG", func(m *models.Message) { m.MediaType, m.Body, m.MediaUrl = "image", "legenda", "/public/a.jpg" })

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/message/:messageId/edit", mc.EditMessage)
	post := func(id, body string) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(map[string]string{"body": body})
		req := httptest.NewRequest(http.MethodPost, "/message/"+id+"/edit", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("edita: manda o comando ao engine, grava a versão como 'me' e avisa a tela", func(t *testing.T) {
		w := post("OK", "  texto novo  ")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		require.Len(t, pub.calls, 1)
		assert.Contains(t, pub.calls[0]["routingKey"], "message.edit")
		cmd := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		assert.Equal(t, "OK", cmd["targetMsgId"], "o alvo é a mensagem original")
		assert.Equal(t, "texto novo", cmd["body"], "o texto sai sem espaços nas pontas")
		assert.Equal(t, "5511999999999", cmd["to"])

		chain, err := services.MessageEdits(t.Context(), db, tenantID, "OK")
		require.NoError(t, err)
		require.Len(t, chain, 2)
		assert.Equal(t, []string{"texto antigo", "texto novo"}, []string{chain[0].Body, chain[1].Body})
		assert.Equal(t, models.EditedByMe, chain[1].EditedBy)
		assert.Len(t, bc.payloads, 1, "a tela é avisada na hora")
	})

	t.Run("legenda de mídia: o comando leva o tipo para o engine trocar a legenda", func(t *testing.T) {
		before := len(pub.calls)
		require.Equal(t, http.StatusOK, post("IMG", "legenda nova").Code)
		cmd := pub.calls[before]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		assert.Equal(t, "image", cmd["mediaType"])
	})

	t.Run("recusas não enviam nada ao engine nem gravam versão", func(t *testing.T) {
		before := len(pub.calls)
		assert.Equal(t, http.StatusUnprocessableEntity, post("VELHA", "x").Code, "fora dos 20 min")
		assert.Equal(t, http.StatusUnprocessableEntity, post("RECEBIDA", "x").Code, "não é nossa")
		assert.Equal(t, http.StatusNotFound, post("NAO_EXISTE", "x").Code)
		assert.Equal(t, http.StatusBadRequest, post("OK", "   ").Code, "texto vazio")
		assert.Equal(t, http.StatusUnprocessableEntity, post("OK", "texto novo").Code, "igual ao atual: não é edição")
		assert.Len(t, pub.calls, before)
		chain, _ := services.MessageEdits(t.Context(), db, tenantID, "VELHA")
		assert.Empty(t, chain)
	})
}
