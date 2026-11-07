package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A resposta rápida é enviada pelo canal da conexão do ticket (RichSender) e a mensagem fica gravada com o escopo da empresa.
func TestQuickAnswerSend_PublicaPeloCanalEGravaAMensagem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &stubPublisher{}
	useWhatsAppChannel(pub)

	wa := models.WhatsAppView{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: wa.ID}
	require.NoError(t, db.Create(&ticket).Error)
	qa := models.QuickAnswer{Shortcut: "/enq", Message: "Gostou?", Type: "poll", TenantID: tenant,
		Content: `{"question":"Gostou?","options":["Sim","Nao"],"maxSelections":1}`}
	require.NoError(t, db.Create(&qa).Error)

	ctrl := NewQuickAnswerController(pub, nil, db, nil)
	r := gin.New()
	r.Use(testScopedMiddleware(db, tenant.String()))
	r.POST("/quickAnswers/:quickAnswerId/send", ctrl.Send)

	req := httptest.NewRequest(http.MethodPost, "/quickAnswers/"+itoa(qa.ID)+"/send", bytes.NewBufferString(`{"ticketId":`+itoa(ticket.ID)+`}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, pub.calls, 1)
	assert.Contains(t, pub.calls[0]["routingKey"], "."+itoa(wa.ID)+".message.send.poll")
	payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
	assert.Equal(t, wa.ID, payload["sessionId"], "o sessionId é o desta conexão")

	var sent models.Message
	require.NoError(t, db.Where(`"ticketId" = ? AND "fromMe" = ?`, ticket.ID, true).First(&sent).Error)
	// O voto do contato e o cartão da tela leem a enquete em dataJson.rich, o mesmo formato da enquete recebida.
	assert.Equal(t, "poll", sent.MediaType)
	options, selectable := pollOptions(sent.DataJson)
	assert.Equal(t, []string{"Sim", "Nao"}, options)
	assert.Equal(t, 1, selectable)
}
