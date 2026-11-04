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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Visualização única só existe no WhatsApp. Num canal agnóstico (Telegram) o pedido é RECUSADO com mensagem clara, e nada
// é enviado nem gravado: mandar como mídia comum enganaria o operador.
func TestSendMessage_ViewOnce_CanalSemSuporteRecusa(t *testing.T) {
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })

	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-VO")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("viewOnce", "true"))
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="medias"; filename="a.jpg"`)
	h.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, _ = part.Write([]byte("\xff\xd8\xff\xe0FOTO"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(tk.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	e.r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "view_once_not_supported")
	assert.Empty(t, e.ch.got, "nada foi enviado ao canal")
	var n int64
	e.db.Table(`"Messages"`).Where(`"tenantId" = ?`, e.tenantID).Count(&n)
	assert.Zero(t, n, "nada foi gravado")
}
