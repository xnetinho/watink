package controllers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// capturingChannel devolve sempre o mesmo id externo (como o Telegram, cujo message_id é por chat) e guarda o que recebeu.
type capturingChannel struct {
	dummyChannel
	externalID string
	got        []domain.OutboundMessage
}

func (c *capturingChannel) Send(_ context.Context, _ *models.ChannelConnection, m domain.OutboundMessage) (domain.SendResult, error) {
	c.got = append(c.got, m)
	return domain.SendResult{ExternalMessageID: c.externalID, Status: "sent"}, nil
}

type sendEnv struct {
	db       *gorm.DB
	tenantID uuid.UUID
	conn     models.ChannelConnection
	r        *gin.Engine
	ch       *capturingChannel
	userID   int
}

func newSendEnv(t *testing.T) *sendEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Tenant Envio"}).Error)
	user := models.User{Name: "Atendente", Email: "a@envio.test", PasswordHash: "h", TenantID: tenantID}
	require.NoError(t, db.Create(&user).Error)

	ch := &capturingChannel{dummyChannel: dummyChannel{kind: domain.ChannelKindTelegram, caps: domain.Capabilities{InitiateConversation: true}}, externalID: "42"}
	domain.DefaultChannelRegistry().Register(ch)

	conn := models.ChannelConnection{TenantID: tenantID, Kind: string(domain.ChannelKindTelegram), Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)

	mc := controllers.NewMessageController(nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID.String())
		c.Set("userId", float64(user.ID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.POST("/messages/:ticketId", mc.SendMessage)
	return &sendEnv{db: db, tenantID: tenantID, conn: conn, r: r, ch: ch, userID: user.ID}
}

// newTicket cria contato e ticket do canal. A linha em Whatsapps existe porque o envio ainda consulta essa tabela
// antes de despachar pelo canal (removida na tarefa 4.3).
func (e *sendEnv) newTicket(t *testing.T, chat string) models.Ticket {
	t.Helper()
	wa := models.Whatsapp{Name: "legacy-" + chat, TenantID: e.tenantID, Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&wa).Error)
	contact := models.Contact{Name: chat, Number: chat, TenantID: e.tenantID}
	require.NoError(t, e.db.Create(&contact).Error)
	ticket := models.Ticket{TenantID: e.tenantID, ContactID: contact.ID, WhatsappID: wa.ID, ChannelConnectionID: &e.conn.ID, ConversationKey: chat, Status: "open"}
	require.NoError(t, e.db.Create(&ticket).Error)
	return ticket
}

func (e *sendEnv) send(t *testing.T, ticketID int, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("POST", "/messages/"+strconv.Itoa(ticketID), bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestSendMessage_MesmoIdExternoEmDoisChatsGravaAsDuas(t *testing.T) {
	e := newSendEnv(t)
	a, b := e.newTicket(t, "chat-A"), e.newTicket(t, "chat-B")

	require.Equal(t, http.StatusOK, e.send(t, a.ID, `{"body":"para A"}`).Code)
	require.Equal(t, http.StatusOK, e.send(t, b.ID, `{"body":"para B"}`).Code)

	var msgs []models.Message
	require.NoError(t, e.db.Where(`"tenantId" = ?`, e.tenantID).Order("body").Find(&msgs).Error)
	require.Len(t, msgs, 2, "a segunda resposta não pode ser descartada pelo id repetido do canal")
	assert.Equal(t, "para A", msgs[0].Body)
	assert.Equal(t, a.ID, msgs[0].TicketID)
	assert.Equal(t, "para B", msgs[1].Body)
	assert.Equal(t, b.ID, msgs[1].TicketID)
	assert.Equal(t, "42", msgs[0].ExternalID)
	assert.NotEqual(t, msgs[0].ID, msgs[1].ID)
}

func TestSendMessage_CitacaoVaiAoCanalComIdExternal(t *testing.T) {
	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-A")
	storedID := domain.StorageMessageID(domain.ChannelKindTelegram, e.conn.ID, "chat-A", "99")
	require.NoError(t, e.db.Create(&models.Message{ID: storedID, ExternalID: "99", Body: "pergunta", TicketID: tk.ID, TenantID: e.tenantID}).Error)

	w := e.send(t, tk.ID, `{"body":"resposta","quotedMsg":{"id":"`+storedID+`"}}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, e.ch.got, 1)
	assert.Equal(t, "99", e.ch.got[0].QuotedMessageID, "o Telegram só entende o message_id dele, não o id interno")
}

func (c *capturingChannel) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	if contact != nil && contact.Number != "" {
		return contact.Number, nil
	}
	return "mock_addr", nil
}
