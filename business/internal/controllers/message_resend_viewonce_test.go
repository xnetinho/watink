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

// Mídia enviada como visualização única que falhou volta como visualização única no reenvio (antes voltava como mídia comum).
func TestResendMessage_MantemVisualizacaoUnica(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	require.NoError(t, db.Create(&ticket).Error)

	mk := func(id, mediaType, data string) {
		require.NoError(t, db.Create(&models.Message{ID: id, MediaType: mediaType, MediaUrl: "/public/media/x", DataJson: data,
			Ack: 5, FromMe: true, TicketID: ticket.ID, TenantID: tenantID}).Error)
	}
	mk("VO_IMG", "image", `{"mimetype":"image/jpeg","viewOnce":true}`)
	mk("VO_VID", "video", `{"mimetype":"video/mp4","viewOnce":true}`)
	mk("VO_AUD", "audio", `{"mimetype":"audio/ogg","viewOnce":true}`)
	mk("COMUM", "image", `{"mimetype":"image/jpeg"}`)

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/message/:messageId/resend", mc.ResendMessage)

	resend := func(id string) (map[string]interface{}, models.Message) {
		pub.calls = nil
		req := httptest.NewRequest(http.MethodPost, "/message/"+id+"/resend", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Len(t, pub.calls, 1)
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		var m models.Message
		require.NoError(t, db.Where(`id = ?`, payload["messageId"]).Take(&m).Error)
		return payload, m
	}

	for _, id := range []string{"VO_IMG", "VO_VID", "VO_AUD"} {
		payload, m := resend(id)
		assert.Equal(t, true, payload["viewOnce"], id+": o comando ao engine leva a marca")
		assert.True(t, dataFieldBool(m.DataJson, "viewOnce"), id+": a nova mensagem guarda a marca (selo na tela)")
	}

	payload, m := resend("COMUM")
	assert.NotContains(t, payload, "viewOnce", "mídia comum continua comum")
	assert.False(t, dataFieldBool(m.DataJson, "viewOnce"))
}
