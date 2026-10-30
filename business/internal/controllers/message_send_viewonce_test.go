package controllers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sendMultipart envia um arquivo pelo compositor (multipart), como o painel de arrastar e soltar faz.
func sendMultipart(t *testing.T, r *gin.Engine, ticketID int, mime, name string, content []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="medias"; filename="`+name+`"`)
	h.Set("Content-Type", mime)
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, _ = part.Write(content)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticketID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestSendMessage_ViewOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// o mediastore grava em public/media relativo ao diretório de trabalho: não pode sujar o repositório
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })

	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
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
	jpeg := []byte("\xff\xd8\xff\xe0FOTO")

	t.Run("foto, vídeo e áudio com viewOnce: o comando leva a marca e a mensagem guarda para o selo", func(t *testing.T) {
		for _, c := range []struct{ mime, name, kind string }{
			{"image/jpeg", "a.jpg", "image"}, {"video/mp4", "a.mp4", "video"}, {"audio/ogg", "a.ogg", "audio"},
		} {
			pub.calls = nil
			rec := sendMultipart(t, r, ticket.ID, c.mime, c.name, jpeg, map[string]string{"viewOnce": "true"})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			payload, m := last()
			assert.Equal(t, true, payload["viewOnce"], c.kind)
			assert.Equal(t, c.kind, m.MediaType)
			var d map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(m.DataJson), &d))
			assert.Equal(t, true, d["viewOnce"], "o dataJson guarda a marca: a tela mostra o selo na mensagem que ENVIAMOS")
		}
	})

	t.Run("sem o campo: mídia comum, nada de viewOnce", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "image/jpeg", "b.jpg", jpeg, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		payload, m := last()
		assert.NotContains(t, payload, "viewOnce")
		assert.NotContains(t, m.DataJson, "viewOnce")
	})

	t.Run("documento e figurinha ignoram o pedido (não têm visualização única) e saem como mídia comum", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "application/pdf", "c.pdf", []byte("%PDF-1.4"), map[string]string{"viewOnce": "true"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		payload, m := last()
		assert.NotContains(t, payload, "viewOnce")
		assert.NotContains(t, m.DataJson, "viewOnce")
	})

	t.Run("viewOnce diferente de true não liga nada", func(t *testing.T) {
		pub.calls = nil
		rec := sendMultipart(t, r, ticket.ID, "image/jpeg", "d.jpg", jpeg, map[string]string{"viewOnce": "false"})
		require.Equal(t, http.StatusOK, rec.Code)
		payload, _ := last()
		assert.NotContains(t, payload, "viewOnce")
	})
}

func TestCanBeViewOnce(t *testing.T) {
	for _, ok := range []string{"image", "video", "audio"} {
		assert.True(t, canBeViewOnce(ok), ok)
	}
	for _, no := range []string{"document", "sticker", "chat", "", "voice_note"} {
		assert.False(t, canBeViewOnce(no), no)
	}
}
