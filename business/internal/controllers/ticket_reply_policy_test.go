package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyChannel struct {
	kind domain.ChannelKind
	caps domain.Capabilities
}

func (d *dummyChannel) Kind() domain.ChannelKind { return d.kind }
func (d *dummyChannel) Capabilities() domain.Capabilities { return d.caps }
func (d *dummyChannel) Connect(_ context.Context, _ *models.ChannelConnection, _ domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{}, nil
}
func (d *dummyChannel) Disconnect(_ context.Context, _ *models.ChannelConnection) error { return nil }
func (d *dummyChannel) Send(_ context.Context, _ *models.ChannelConnection, _ domain.OutboundMessage) (domain.SendResult, error) {
	return domain.SendResult{}, nil
}
func (d *dummyChannel) ParseInbound(_ context.Context, _ *models.ChannelConnection, _ []byte, _ map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

func TestTicketReplyPolicy_ShowAndSend409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)

	tenantID := uuid.New()
	tenant := models.Tenant{ID: tenantID, Name: "Tenant Reply Policy"}
	require.NoError(t, db.Create(&tenant).Error)

	user := models.User{
		Name:         "Atendente Teste",
		Email:        "atendente@watink.test",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}
	require.NoError(t, db.Create(&user).Error)

	contact := models.Contact{
		Name:     "Contato Instagram",
		Number:   "ig_12345",
		TenantID: tenantID,
	}
	require.NoError(t, db.Create(&contact).Error)

	// Registra canal Instagram com janela de 24h
	window24h := 24 * time.Hour
	domain.DefaultChannelRegistry().Register(&dummyChannel{
		kind: domain.ChannelKindInstagram,
		caps: domain.Capabilities{
			ReplyWindow:          &window24h,
			InitiateConversation: false,
		},
	})

	conn := models.ChannelConnection{
		TenantID: tenantID,
		Kind:     string(domain.ChannelKindInstagram),
		Name:     "Insta Direct",
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&conn).Error)

	// Ticket com janela de resposta expirada (última mensagem do cliente foi há 30h)
	ticket := models.Ticket{
		TenantID:            tenantID,
		ContactID:           contact.ID,
		ChannelConnectionID: &conn.ID,
		ConversationKey:     "ig_12345",
		Status:              "open",
	}
	require.NoError(t, db.Create(&ticket).Error)

	oldInbound := models.Message{
		ID:        "msg-old",
		Body:      "Oi",
		TicketID:  ticket.ID,
		FromMe:    false,
		TenantID:  tenantID,
		CreatedAt: time.Now().Add(-30 * time.Hour),
	}
	require.NoError(t, db.Create(&oldInbound).Error)

	// Setup controllers
	tc := controllers.NewTicketController(nil, nil, nil, nil)
	mc := controllers.NewMessageController(nil, nil, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID.String())
		c.Set("userId", float64(user.ID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/tickets/:ticketId", tc.ShowTicket)
	r.POST("/messages/:ticketId", mc.SendMessage)

	// 1. Testa GET /tickets/:id devolvendo replyPolicy expirada
	wShow := httptest.NewRecorder()
	reqShow, _ := http.NewRequest("GET", "/tickets/"+strconv.Itoa(ticket.ID), nil)
	r.ServeHTTP(wShow, reqShow)

	require.Equal(t, http.StatusOK, wShow.Code)
	var showBody map[string]interface{}
	require.NoError(t, json.Unmarshal(wShow.Body.Bytes(), &showBody))
	assert.Contains(t, showBody, "replyPolicy")
	rp := showBody["replyPolicy"].(map[string]interface{})
	assert.Equal(t, false, rp["canReply"])
	assert.Equal(t, string(domain.ReplyBlockWindowExpired), rp["reason"])

	// 2. Testa POST /messages/:ticketId recusando com 409 Conflict
	payload := []byte(`{"body":"Tentando responder fora da janela"}`)
	wSend := httptest.NewRecorder()
	reqSend, _ := http.NewRequest("POST", "/messages/"+strconv.Itoa(ticket.ID), bytes.NewBuffer(payload))
	reqSend.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wSend, reqSend)

	assert.Equal(t, http.StatusConflict, wSend.Code)
	var sendErr map[string]interface{}
	require.NoError(t, json.Unmarshal(wSend.Body.Bytes(), &sendErr))
	assert.Equal(t, "reply_not_allowed", sendErr["error"])
	assert.Equal(t, string(domain.ReplyBlockWindowExpired), sendErr["code"])
}
