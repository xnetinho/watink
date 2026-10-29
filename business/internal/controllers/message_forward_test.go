package controllers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// contactOf cria o contato de um chat do Telegram NESTA conexão (identidade incluída), como se ele tivesse sido visto antes.
func (e *sendEnv) contactOf(t *testing.T, chat string) models.Contact {
	t.Helper()
	ct := models.Contact{Name: "Contato " + chat, Number: "", TenantID: e.tenantID, ConnectionSource: "channel", ConnectionID: e.conn.ID}
	require.NoError(t, e.db.Create(&ct).Error)
	require.NoError(t, e.db.Create(&models.ContactIdentity{TenantID: e.tenantID, ContactID: ct.ID, Channel: "telegram", ConnectionID: e.conn.ID, ExternalID: chat, Meta: "{}"}).Error)
	return ct
}

func (e *sendEnv) seedMessage(t *testing.T, tk models.Ticket, id, ext, body, mediaType, mediaURL string) string {
	t.Helper()
	sid := domain.StorageMessageID(domain.ChannelKindTelegram, e.conn.ID, tk.ConversationKey, ext)
	require.NoError(t, e.db.Create(&models.Message{ID: sid, ExternalID: ext, Body: body, MediaType: mediaType, MediaUrl: mediaURL, TicketID: tk.ID, TenantID: e.tenantID, DataJson: `{"mimetype":"image/png"}`}).Error)
	return sid
}

func ids(cs ...models.Contact) string {
	out := ""
	for i, c := range cs {
		if i > 0 {
			out += ","
		}
		out += itoa(c.ID)
	}
	return out
}

func TestForward_ParaContatoQueNuncaConversouCriaAConversaNaConexaoDele(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "olá mundo", "", "")

	w, body := e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	require.Equal(t, http.StatusOK, w.Code, body)
	assert.Contains(t, body, `"ok":true`)

	require.Len(t, e.ch.got, 1)
	assert.Equal(t, "olá mundo", e.ch.got[0].Body)
	assert.Equal(t, "chat-B", e.ch.got[0].ConversationKey, "a mensagem vai ao chat do contato escolhido")
	require.NotNil(t, e.ch.got[0].Forward)
	assert.True(t, e.ch.got[0].Forward.AsNew)

	var t2 models.Ticket
	require.NoError(t, e.db.Where(`"contactId" = ?`, dest.ID).First(&t2).Error)
	require.NotNil(t, t2.ChannelConnectionID)
	assert.Equal(t, e.conn.ID, *t2.ChannelConnectionID, "a conversa nasce na conexão do contato")
	var saved models.Message
	require.NoError(t, e.db.Where(`"fromMe" = true AND "ticketId" = ?`, t2.ID).First(&saved).Error)
	assert.JSONEq(t, `{"mimetype":"image/png"}`, saved.DataJson, "cópia não leva o selo")
}

func TestForward_ReaproveitaAConversaAbertaDoContato(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	open := models.Ticket{TenantID: e.tenantID, ContactID: dest.ID, ChannelConnectionID: &e.conn.ID, ConversationKey: "chat-B", Status: "open"}
	require.NoError(t, e.db.Create(&open).Error)
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")

	_, body := e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	assert.Contains(t, body, `"ticketId":`+itoa(open.ID))
	var n int64
	e.db.Model(&models.Ticket{}).Where(`"contactId" = ?`, dest.ID).Count(&n)
	assert.EqualValues(t, 1, n, "não abre uma segunda conversa")
}

func TestForward_DeVerdadeGravaSeloEEnviaReferenciaDaOriginal(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	e.ch.caps.Forward = true

	w, body := e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"forward"}`)
	require.Equal(t, http.StatusOK, w.Code, body)

	require.Len(t, e.ch.got, 1)
	f := e.ch.got[0].Forward
	require.NotNil(t, f)
	assert.False(t, f.AsNew)
	assert.True(t, f.SameConnection, "origem e destino são a mesma conexão")
	assert.Equal(t, "10", f.ExternalMessageID, "o canal encaminha pelo id dele, não pelo id interno")
	assert.Equal(t, "chat-A", f.FromConversationKey)

	var saved models.Message
	require.NoError(t, e.db.Where(`"fromMe" = true AND "ticketId" = ?`, e.ticketOf(t, dest).ID).First(&saved).Error)
	assert.JSONEq(t, `{"mimetype":"image/png","forwarded":true,"forwardScore":1}`, saved.DataJson)
}

func (e *sendEnv) ticketOf(t *testing.T, c models.Contact) models.Ticket {
	t.Helper()
	var tk models.Ticket
	require.NoError(t, e.db.Where(`"contactId" = ?`, c.ID).First(&tk).Error)
	return tk
}

func TestForward_ContatoDeOutraConexaoDoMesmoCanalUsaAConexaoDele(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	e.ch.caps.Forward = true

	other := models.ChannelConnection{TenantID: e.tenantID, Kind: "telegram", Name: "Outro bot", Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&other).Error)
	ct := models.Contact{Name: "Zé", TenantID: e.tenantID, ConnectionSource: "channel", ConnectionID: other.ID}
	require.NoError(t, e.db.Create(&ct).Error)
	require.NoError(t, e.db.Create(&models.ContactIdentity{TenantID: e.tenantID, ContactID: ct.ID, Channel: "telegram", ConnectionID: other.ID, ExternalID: "chat-Z", Meta: "{}"}).Error)

	_, body := e.forward(t, id, `{"contactIds":[`+ids(ct)+`],"mode":"copy"}`)
	assert.Contains(t, body, `"ok":true`)
	require.NotNil(t, e.ticketOf(t, ct).ChannelConnectionID)
	assert.Equal(t, other.ID, *e.ticketOf(t, ct).ChannelConnectionID, "sai pela conexão do contato, não pela da mensagem de origem")

	_, body = e.forward(t, id, `{"contactIds":[`+ids(ct)+`],"mode":"forward"}`)
	assert.Contains(t, body, `"code":"forward_not_supported"`, "encaminhar com selo exige a MESMA conexão; a cópia continua possível")
}

func TestForward_CanalSemSuporteRecusaOModoDeVerdadeMasCopia(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	e.ch.caps.Forward = false

	_, body := e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"forward"}`)
	assert.Contains(t, body, `"code":"forward_not_supported"`)
	assert.Empty(t, e.ch.got)

	_, body = e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	assert.Contains(t, body, `"ok":true`)
}

func TestForward_EntreCanaisDiferentesERecusado(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	wa := models.Whatsapp{Name: "wa", TenantID: e.tenantID, Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&wa).Error)
	ct := models.Contact{Name: "Zé", Number: "5511999", TenantID: e.tenantID, ConnectionSource: "whatsapp", ConnectionID: wa.ID}
	require.NoError(t, e.db.Create(&ct).Error)

	for _, mode := range []string{"copy", "forward"} {
		_, body := e.forward(t, id, `{"contactIds":[`+ids(ct)+`],"mode":"`+mode+`"}`)
		assert.Contains(t, body, `"code":"different_channel"`, mode)
	}
	assert.Empty(t, e.ch.got, "nada pode sair para outro canal")
}

func TestForward_ContatoSemConexaoValidaOuDeOutraEmpresaNaoRecebe(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")
	orphan := models.Contact{Name: "Órfão", Number: "1", TenantID: e.tenantID, ConnectionSource: "channel", ConnectionID: 99999}
	require.NoError(t, e.db.Create(&orphan).Error)
	alheio := models.Contact{Name: "Alheio", TenantID: uuid.New(), ConnectionSource: "channel", ConnectionID: e.conn.ID}
	require.NoError(t, e.db.Create(&alheio).Error)

	_, body := e.forward(t, id, `{"contactIds":[`+ids(orphan, alheio)+`],"mode":"copy"}`)
	assert.Contains(t, body, `"code":"connection_required"`)
	assert.Contains(t, body, `"code":"contact_not_found"`, "contato de outra empresa não existe para esta")
	assert.Empty(t, e.ch.got)
}

func TestForward_TiposQueNaoTemEquivalenteESemMidiaBaixada(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	poll := e.seedMessage(t, from, "p", "11", "📊 enquete", "poll", "")
	pending := e.seedMessage(t, from, "i", "12", "", "image", "")

	w, body := e.forward(t, poll, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, body, "message_not_forwardable")

	w, body = e.forward(t, pending, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, body, "media_unavailable")
	assert.Empty(t, e.ch.got)
}

func TestForward_MidiaVaiComArquivoEMimetype(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")
	id := e.seedMessage(t, from, "i", "12", "legenda", "image", "/public/media/x.png")

	_, body := e.forward(t, id, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	require.Contains(t, body, `"ok":true`)
	require.Len(t, e.ch.got, 1)
	assert.Equal(t, "image", e.ch.got[0].MediaType)
	assert.Equal(t, "/public/media/x.png", e.ch.got[0].MediaURL)
	assert.Equal(t, "legenda", e.ch.got[0].Body)
}

func TestForward_VariosDestinosEntradaInvalida(t *testing.T) {
	e := newSendEnv(t)
	from := e.newTicket(t, "chat-A")
	b, c := e.contactOf(t, "chat-B"), e.contactOf(t, "chat-C")
	id := e.seedMessage(t, from, "m1", "10", "oi", "", "")

	_, body := e.forward(t, id, `{"contactIds":[`+ids(b, c, b)+`,999999],"mode":"copy"}`)
	assert.Len(t, e.ch.got, 2, "repetido conta uma vez; inexistente não envia")
	assert.Contains(t, body, `"code":"contact_not_found"`)

	w, _ := e.forward(t, id, `{"contactIds":[],"mode":"copy"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w, _ = e.forward(t, id, `{"contactIds":[1],"mode":"xpto"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w, _ = e.forward(t, "nao-existe", `{"contactIds":[1],"mode":"copy"}`)
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
	assert.Equal(t, []int{tg.ID}, ids("channel=telegram&searchParam=chat-A"), "busca por nome do contato: antes respondia 500 (tenantId ambíguo)")
	assert.Empty(t, ids("channel=telegram&searchParam=zzzz"))
}

func TestSendMessage_MidiaGravaOMimetypeParaPoderSerEncaminhada(t *testing.T) {
	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-A")
	dest := e.contactOf(t, "chat-B")

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="medias"; filename="a.png"`)
	hdr.Set("Content-Type", "image/png")
	fw, _ := mw.CreatePart(hdr)
	_, _ = fw.Write([]byte("\x89PNG\r\n\x1a\n"))
	_ = mw.Close()
	req, _ := http.NewRequest("POST", "/messages/"+itoa(tk.ID), body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var sent models.Message
	require.NoError(t, e.db.Where(`"fromMe" = true AND "ticketId" = ? AND "mediaType" = 'image'`, tk.ID).First(&sent).Error)
	assert.JSONEq(t, `{"mimetype":"image/png","fileName":"a.png"}`, sent.DataJson, "guarda mimetype e o nome original do arquivo")

	_, fb := e.forward(t, sent.ID, `{"contactIds":[`+ids(dest)+`],"mode":"copy"}`)
	require.Contains(t, fb, `"ok":true`)
	require.Len(t, e.ch.got, 2)
	assert.Equal(t, "image", e.ch.got[1].MediaType)
}
