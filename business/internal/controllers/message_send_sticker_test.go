package controllers

import (
	"net/http"
	"os"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage_Sticker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })

	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)
	useWhatsAppChannel(pub)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	require.NoError(t, db.Create(&ticket).Error)

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/messages/:ticketId", mc.SendMessage)

	last := func() (map[string]interface{}, models.Message) {
		payload := pub.calls[len(pub.calls)-1]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		var m models.Message
		require.NoError(t, db.Where(`id = ?`, payload["messageId"]).Take(&m).Error)
		return payload, m
	}
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")

	t.Run("webp com o pedido sai como figurinha e sem legenda", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "image/webp", "f.webp", webp, map[string]string{"sticker": "true", "body": "legenda que não pode sair"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		payload, m := last()
		assert.Equal(t, "sticker", payload["mediaType"])
		assert.Equal(t, "sticker", m.MediaType)
		assert.Empty(t, m.Body, "figurinha não tem legenda")
		assert.Empty(t, payload["body"])
	})

	t.Run("webp sem o pedido continua foto", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "image/webp", "g.webp", webp, map[string]string{"body": "foto"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		payload, m := last()
		assert.Equal(t, "image", payload["mediaType"])
		assert.Equal(t, "image", m.MediaType)
		assert.Equal(t, "foto", m.Body)
	})

	t.Run("PNG, JPG, GIF e vídeo com o pedido também viram figurinha (o engine converte)", func(t *testing.T) {
		for _, c := range []struct{ mime, name string }{{"image/png", "a.png"}, {"image/jpeg", "a.jpg"}, {"image/gif", "a.gif"}, {"video/mp4", "a.mp4"}} {
			pub.calls = nil
			rec := sendMultipart(t, r, ticket.ID, c.mime, c.name, []byte("\xff\xd8\xff\xe0DADOS"), map[string]string{"sticker": "true", "body": "x"})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			payload, m := last()
			assert.Equal(t, "sticker", payload["mediaType"], c.mime)
			assert.Equal(t, "sticker", m.MediaType, c.mime)
			assert.Empty(t, m.Body, c.mime)
		}
	})

	t.Run("o pedido em documento ou áudio é ignorado: sai como o que é", func(t *testing.T) {
		for _, c := range []struct{ mime, name, want string }{{"application/pdf", "a.pdf", "document"}, {"audio/mpeg", "a.mp3", "audio"}} {
			pub.calls = nil
			rec := sendMultipart(t, r, ticket.ID, c.mime, c.name, []byte("%PDF-1.4"), map[string]string{"sticker": "true"})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			payload, _ := last()
			assert.Equal(t, c.want, payload["mediaType"], c.mime)
		}
	})

	t.Run("sticker diferente de true não liga nada", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "image/webp", "i.webp", webp, map[string]string{"sticker": "false"})
		require.Equal(t, http.StatusOK, rec.Code)
		payload, _ := last()
		assert.Equal(t, "image", payload["mediaType"])
	})
}
