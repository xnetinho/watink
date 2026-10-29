package controllers

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestQuotedMediaOf(t *testing.T) {
	t.Run("voz recebida do celular leva microfone e duração", func(t *testing.T) {
		got := quotedMediaOf(models.Message{MediaType: "audio", DataJson: `{"mimetype":"audio/ogg; codecs=opus","rich":{"voice":true,"seconds":4}}`})
		assert.Equal(t, map[string]interface{}{"type": "audio", "mimeType": "audio/ogg; codecs=opus", "voice": true, "seconds": 4}, got)
	})
	t.Run("voz enviada pelo Watink", func(t *testing.T) {
		got := quotedMediaOf(models.Message{MediaType: "audio", DataJson: `{"voice":true}`})
		assert.Equal(t, true, got["voice"])
	})
	t.Run("arquivo de áudio não vira voz", func(t *testing.T) {
		got := quotedMediaOf(models.Message{MediaType: "audio", DataJson: `{"mimetype":"audio/mpeg","voice":false}`})
		assert.NotContains(t, got, "voice")
	})
	t.Run("imagem leva miniatura e legenda", func(t *testing.T) {
		got := quotedMediaOf(models.Message{MediaType: "image", Body: "olha", DataJson: `{"thumbnail":"QUJD","mimetype":"image/jpeg"}`})
		assert.Equal(t, "QUJD", got["thumbnail"])
		assert.Equal(t, "olha", got["caption"])
	})
	t.Run("texto e dataJson inválido não quebram", func(t *testing.T) {
		assert.Nil(t, quotedMediaOf(models.Message{MediaType: "", Body: "oi"}))
		assert.Equal(t, "audio", quotedMediaOf(models.Message{MediaType: "audio", DataJson: "{x"})["type"])
	})
}
