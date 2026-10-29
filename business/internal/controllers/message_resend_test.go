package controllers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *sendEnv) resend(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("POST", "/message/"+id+"/resend", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.forwardRouter().ServeHTTP(w, req)
	return w
}

func TestResendMessage(t *testing.T) {
	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-R")

	mk := func(id, body, mediaType, mediaURL, data string, ack int, fromMe bool) {
		require.NoError(t, e.db.Create(&models.Message{ID: id, Body: body, MediaType: mediaType, MediaUrl: mediaURL, DataJson: data,
			Ack: ack, FromMe: fromMe, TicketID: tk.ID, TenantID: e.tenantID}).Error)
	}
	mk("TEXTO", "oi", "", "", "{}", 5, true)
	mk("VOZ", "", "audio", "/public/media/a.ogg", `{"voice":true,"mimetype":"audio/ogg"}`, 5, true)
	mk("ARQUIVO", "", "audio", "/public/media/b.mp3", `{"voice":false,"mimetype":"audio/mpeg","fileName":"b.mp3"}`, 5, true)
	mk("CELULAR_OFF", "x", "", "", "{}", 1, true)
	mk("ENTREGUE", "x", "", "", "{}", 2, true)
	mk("RECEBIDA", "x", "", "", "{}", 5, false)
	mk("ENQUETE", "x", "poll", "", "{}", 5, true)
	mk("SEM_ARQUIVO", "", "image", "", "{}", 5, true)

	t.Run("texto que falhou é reenviado; a falha antiga continua no histórico", func(t *testing.T) {
		before := len(e.ch.got)
		w := e.resend(t, "TEXTO")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Len(t, e.ch.got, before+1)
		assert.Equal(t, "oi", e.ch.got[before].Body)
		var old models.Message
		require.NoError(t, e.db.Where("id = ?", "TEXTO").Take(&old).Error)
		assert.False(t, old.IsDeleted)
		assert.Equal(t, 5, old.Ack)
	})

	t.Run("mídia volta com o mesmo arquivo (o botão antigo mandava só o corpo, vazio)", func(t *testing.T) {
		before := len(e.ch.got)
		require.Equal(t, http.StatusOK, e.resend(t, "VOZ").Code)
		require.Len(t, e.ch.got, before+1)
		assert.Equal(t, "audio", e.ch.got[before].MediaType)
		assert.Equal(t, "/public/media/a.ogg", e.ch.got[before].MediaURL)

		require.Equal(t, http.StatusOK, e.resend(t, "ARQUIVO").Code)
		assert.Equal(t, "/public/media/b.mp3", e.ch.got[before+1].MediaURL)
	})

	t.Run("só reenvia o que falhou: ack 1 (celular desligado) e ack 2 duplicariam", func(t *testing.T) {
		before := len(e.ch.got)
		for _, id := range []string{"CELULAR_OFF", "ENTREGUE"} {
			w := e.resend(t, id)
			assert.Equal(t, http.StatusConflict, w.Code, id)
			assert.Contains(t, w.Body.String(), "message_not_failed")
		}
		assert.Len(t, e.ch.got, before, "nada pode ter sido enviado")
	})

	t.Run("recusa o que não é nosso, não tem como reenviar ou não existe", func(t *testing.T) {
		assert.Equal(t, http.StatusUnprocessableEntity, e.resend(t, "RECEBIDA").Code)
		assert.Equal(t, http.StatusUnprocessableEntity, e.resend(t, "ENQUETE").Code)
		assert.Equal(t, http.StatusUnprocessableEntity, e.resend(t, "SEM_ARQUIVO").Code)
		assert.Equal(t, http.StatusNotFound, e.resend(t, "NAO_EXISTE").Code)
	})
}
