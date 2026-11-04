package controllers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Figurinha pedida num canal que não é WhatsApp é recusada (nada sai e nada é gravado), como a visualização única.
func TestSendMessage_StickerEmCanalAgnosticoERecusada(t *testing.T) {
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })

	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-S")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("sticker", "true"))
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="medias"; filename="f.webp"`)
	h.Set("Content-Type", "image/webp")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, _ = part.Write([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "))
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(tk.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	e.r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "sticker_not_supported")
	assert.Empty(t, e.ch.got, "nada pode ter sido enviado pelo canal")
	var n int64
	require.NoError(t, e.db.Model(&models.Message{}).Where(`"tenantId" = ?`, e.tenantID).Count(&n).Error)
	assert.Zero(t, n, "nada pode ter sido gravado")
}
