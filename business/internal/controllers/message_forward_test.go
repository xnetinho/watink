package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *sendEnv) forwardRouter() *gin.Engine {
	mc := controllers.NewMessageController(nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", e.db)
		c.Set("tenantId", e.tenantID.String())
		c.Set("userId", float64(e.userID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.POST("/message/:messageId/forward", mc.ForwardMessage)
	return r
}

func (e *sendEnv) forward(t *testing.T, msgID, payload string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	req, _ := http.NewRequest("POST", "/message/"+msgID+"/forward", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.forwardRouter().ServeHTTP(w, req)
	return w, w.Body.String()
}

func (e *sendEnv) seedMessage(t *testing.T, tk models.Ticket, id, ext, body, mediaType, mediaURL string) string {
	t.Helper()
	sid := domain.StorageMessageID(domain.ChannelKindTelegram, e.conn.ID, tk.ConversationKey, ext)
	require.NoError(t, e.db.Create(&models.Message{ID: sid, ExternalID: ext, Body: body, MediaType: mediaType, MediaUrl: mediaURL, TicketID: tk.ID, TenantID: e.tenantID, DataJson: `{"mimetype":"image/png"}`}).Error)
	return sid
}

func TestForward_CopiaComoNovaVaiAoMesmoCanalSemSelo(t *testing.T) {
	e := newSendEnv(t)
	from, to := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "olá mundo", "", "")

	w, body := e.forward(t, id, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"copy"}`)
	require.Equal(t, http.StatusOK, w.Code, body)
	assert.Contains(t, body, `"ok":true`)

	require.Len(t, e.ch.got, 1)
	assert.Equal(t, "olá mundo", e.ch.got[0].Body)
	require.NotNil(t, e.ch.got[0].Forward)
	assert.True(t, e.ch.got[0].Forward.AsNew)

	var saved models.Message
	require.NoError(t, e.db.Where(`"fromMe" = true AND "ticketId" = ?`, to.ID).First(&saved).Error)
	assert.JSONEq(t, `{"mimetype":"image/png"}`, saved.DataJson, "cópia não leva o selo")
}

func TestForward_DeVerdadeGravaSeloEEnviaReferenciaDaOriginal(t *testing.T) {
	e := newSendEnv(t)
	from, to := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	e.ch.caps.Forward = true

	w, body := e.forward(t, id, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"forward"}`)
	require.Equal(t, http.StatusOK, w.Code, body)

	require.Len(t, e.ch.got, 1)
	f := e.ch.got[0].Forward
	require.NotNil(t, f)
	assert.False(t, f.AsNew)
	assert.True(t, f.SameConnection)
	assert.Equal(t, "10", f.ExternalMessageID, "o canal encaminha pelo id dele, não pelo id interno")
	assert.Equal(t, "chat-A", f.FromConversationKey)

	var saved models.Message
	require.NoError(t, e.db.Where(`"fromMe" = true AND "ticketId" = ?`, to.ID).First(&saved).Error)
	assert.JSONEq(t, `{"mimetype":"image/png","forwarded":true,"forwardScore":1}`, saved.DataJson)
}

func TestForward_CanalSemSuporteRecusaOModoDeVerdadeMasCopia(t *testing.T) {
	e := newSendEnv(t)
	from, to := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	e.ch.caps.Forward = false

	_, body := e.forward(t, id, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"forward"}`)
	assert.Contains(t, body, `"code":"forward_not_supported"`)
	assert.Empty(t, e.ch.got)

	_, body = e.forward(t, id, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"copy"}`)
	assert.Contains(t, body, `"ok":true`)
}

func TestForward_EntreCanaisDiferentesERecusado(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	// destino sem ChannelConnectionID = WhatsApp
	wa := models.Whatsapp{Name: "wa", TenantID: e.tenantID, Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&wa).Error)
	ct := models.Contact{Name: "Zé", Number: "5511999", TenantID: e.tenantID}
	require.NoError(t, e.db.Create(&ct).Error)
	other := models.Ticket{TenantID: e.tenantID, ContactID: ct.ID, WhatsappID: wa.ID, Status: "open"}
	require.NoError(t, e.db.Create(&other).Error)

	for _, mode := range []string{"copy", "forward"} {
		_, body := e.forward(t, id, `{"ticketIds":[`+itoa(other.ID)+`],"mode":"`+mode+`"}`)
		assert.Contains(t, body, `"code":"different_channel"`, mode)
	}
	assert.Empty(t, e.ch.got, "nada pode sair para outro canal")
}

func TestForward_TiposQueNaoTemEquivalenteESemMidiaBaixada(t *testing.T) {
	e := newSendEnv(t)
	from, to := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")
	poll := e.seedMessage(t, from, "p", "11", "📊 enquete", "poll", "")
	pending := e.seedMessage(t, from, "i", "12", "", "image", "")

	w, body := e.forward(t, poll, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"copy"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, body, "message_not_forwardable")

	w, body = e.forward(t, pending, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"copy"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, body, "media_unavailable")
	assert.Empty(t, e.ch.got)
}

func TestForward_MidiaVaiComArquivoEMimetype(t *testing.T) {
	e := newSendEnv(t)
	from, to := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")
	id := e.seedMessage(t, from, "i", "12", "legenda", "image", "/public/media/x.png")

	_, body := e.forward(t, id, `{"ticketIds":[`+itoa(to.ID)+`],"mode":"copy"}`)
	require.Contains(t, body, `"ok":true`)
	require.Len(t, e.ch.got, 1)
	assert.Equal(t, "image", e.ch.got[0].MediaType)
	assert.Equal(t, "/public/media/x.png", e.ch.got[0].MediaURL)
	assert.Equal(t, "legenda", e.ch.got[0].Body)
}

func TestForward_VariosDestinosEntradaInvalidaETenantAlheio(t *testing.T) {
	e := newSendEnv(t)
	from, b, c := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B"), e.newTicket(t, "chat-C")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")

	_, body := e.forward(t, id, `{"ticketIds":[`+itoa(b.ID)+`,`+itoa(c.ID)+`,`+itoa(b.ID)+`,999999],"mode":"copy"}`)
	assert.Len(t, e.ch.got, 2, "repetido conta uma vez; inexistente não envia")
	assert.Contains(t, body, `"code":"ticket_not_found"`)

	w, _ := e.forward(t, id, `{"ticketIds":[],"mode":"copy"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w, _ = e.forward(t, id, `{"ticketIds":[1],"mode":"xpto"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w, _ = e.forward(t, "nao-existe", `{"ticketIds":[1],"mode":"copy"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListTickets_FiltroPorCanalParaDestinoDeEncaminhamento(t *testing.T) {
	e := newSendEnv(t)
	tg := e.newTicket(t, "chat-A")
	wa := models.Whatsapp{Name: "wa", TenantID: e.tenantID, Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&wa).Error)
	ct := models.Contact{Name: "Zé", Number: "5511999", TenantID: e.tenantID}
	require.NoError(t, e.db.Create(&ct).Error)
	require.NoError(t, e.db.Create(&models.Ticket{TenantID: e.tenantID, ContactID: ct.ID, WhatsappID: wa.ID, Status: "open"}).Error)

	tc := controllers.NewTicketController(nil, nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", e.db)
		c.Set("tenantId", e.tenantID.String())
		c.Set("userId", float64(e.userID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/tickets", tc.ListTickets)
	ids := func(q string) []int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/tickets?isGroup=all&"+q, nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp struct{ Tickets []struct{ ID int } }
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		out := []int{}
		for _, x := range resp.Tickets {
			out = append(out, x.ID)
		}
		return out
	}
	assert.Equal(t, []int{tg.ID}, ids("channel=telegram"), "só o canal pedido")
	assert.Len(t, ids("channel=whatsapp"), 1)
	assert.NotContains(t, ids("channel=whatsapp"), tg.ID)
	assert.Len(t, ids("limit=1"), 1)
}
