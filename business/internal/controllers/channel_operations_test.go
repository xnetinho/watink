package controllers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canalSemOperacoes implementa só o mínimo de domain.Channel: nenhuma operação opcional (reação, edição, voto, presença).
type canalSemOperacoes struct{}

func (canalSemOperacoes) Kind() domain.ChannelKind          { return "semoperacoes" }
func (canalSemOperacoes) Capabilities() domain.Capabilities { return domain.Capabilities{} }
func (canalSemOperacoes) Connect(context.Context, *models.ChannelConnection, domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{}, nil
}
func (canalSemOperacoes) Disconnect(context.Context, *models.ChannelConnection) error { return nil }
func (canalSemOperacoes) Send(context.Context, *models.ChannelConnection, domain.OutboundMessage) (domain.SendResult, error) {
	return domain.SendResult{}, nil
}
func (canalSemOperacoes) ResolveAddress(context.Context, *models.ChannelConnection, *models.Contact) (string, error) {
	return "", nil
}
func (canalSemOperacoes) ParseInbound(context.Context, *models.ChannelConnection, []byte, map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

// Reação, edição, voto e presença são operações opcionais do canal: o WhatsApp as publica ao engine, e um canal que não as tem
// (aqui um canal sem nenhuma) recebe 422 sem publicar nada. Antes a regra era "ticket sem channelConnectionId é WhatsApp".
func TestOperacoesDoCanal_WhatsAppPublicaECanalSemOperacaoRecusa(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)
	useWhatsAppChannel(pub)
	domain.DefaultChannelRegistry().Register(canalSemOperacoes{})

	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	other := models.ChannelConnection{TenantID: tenant, Kind: "semoperacoes", Name: "X", Status: "CONNECTED"}
	require.NoError(t, db.Create(&other).Error)
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	tkWA := models.Ticket{Status: "open", TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: wa.ID}
	require.NoError(t, db.Create(&tkWA).Error)
	tkOther := models.Ticket{Status: "open", TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: other.ID, ConversationKey: "k"}
	require.NoError(t, db.Create(&tkOther).Error)
	for _, m := range []models.Message{
		{ID: "WA-1", Body: "oi", TicketID: tkWA.ID, FromMe: true, Ack: 2, MediaType: "chat", ContactID: &contact.ID, TenantID: tenant, Reactions: "[]"},
		{ID: "OT-1", Body: "oi", TicketID: tkOther.ID, FromMe: true, Ack: 2, MediaType: "chat", ContactID: &contact.ID, TenantID: tenant, Reactions: "[]"},
	} {
		require.NoError(t, db.Create(&m).Error)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenant.String()))
	r.POST("/message/:messageId/react", mc.ReactToMessage)
	r.PUT("/message/:messageId/edit", mc.EditMessage)
	r.POST("/tickets/:ticketId/presence", mc.SubscribePresence)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusOK, do(http.MethodPost, "/message/WA-1/react", `{"reaction":"👍"}`).Code)
	assert.Equal(t, http.StatusOK, do(http.MethodPut, "/message/WA-1/edit", `{"body":"oi editado"}`).Code)
	assert.Equal(t, http.StatusAccepted, do(http.MethodPost, "/tickets/"+itoa(tkWA.ID)+"/presence", ``).Code)
	require.Len(t, pub.calls, 3, "reação, edição e presença chegam ao engine")

	pub.calls = nil
	assert.Equal(t, http.StatusUnprocessableEntity, do(http.MethodPost, "/message/OT-1/react", `{"reaction":"👍"}`).Code)
	assert.Equal(t, http.StatusUnprocessableEntity, do(http.MethodPut, "/message/OT-1/edit", `{"body":"x"}`).Code)
	assert.Equal(t, http.StatusNoContent, do(http.MethodPost, "/tickets/"+itoa(tkOther.ID)+"/presence", ``).Code)
	assert.Empty(t, pub.calls, "canal sem a operação não publica nada")
}
