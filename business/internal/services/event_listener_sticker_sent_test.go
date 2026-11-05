package services_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stkEnv struct {
	bc     *voBroadcast
	tenant uuid.UUID
	tk     models.Ticket
	db     *gorm.DB
	get    func(string) models.Message
	send   func(string, map[string]interface{}) error
}

func stickerSentEnv(t *testing.T) stkEnv {
	t.Helper()
	inTempDir(t)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)
	bc := &voBroadcast{}
	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(db), nil, repository.NewGORMTicketRepo(db), nil, bc, db, nil, nil, nil, nil)
	get := func(id string) models.Message {
		var m models.Message
		require.NoError(t, db.Where("id = ?", id).Take(&m).Error)
		return m
	}
	send := func(id string, extra map[string]interface{}) error {
		p := map[string]interface{}{"messageId": id, "mediaData": base64.StdEncoding.EncodeToString([]byte("RIFF....WEBPVP8 ")), "mimetype": "image/webp", "replaced": true, "error": ""}
		for k, v := range extra {
			p[k] = v
		}
		b, _ := json.Marshal(p)
		return el.HandleMediaDownloadedEvent(context.Background(), b, tenant)
	}
	return stkEnv{bc: bc, tenant: tenant, tk: tk, db: db, get: get, send: send}
}

// A figurinha enviada passa a apontar para o WebP final (512x512) que o destinatário recebeu, e a tela é avisada.
func TestHandleMediaDownloaded_FigurinhaEnviadaTrocaPeloWebPFinal(t *testing.T) {
	e := stickerSentEnv(t)
	// a mensagem enviada com o arquivo ORIGINAL do operador
	require.NoError(t, e.db.Create(&models.Message{ID: "STK1", MediaType: "sticker", MediaUrl: "/public/media/original.png", FromMe: true,
		TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{"fileName":"a.png","mimetype":"image/png"}`}).Error)

	require.NoError(t, e.send("STK1", map[string]interface{}{"animated": true}))

	m := e.get("STK1")
	assert.NotEqual(t, "/public/media/original.png", m.MediaUrl, "passa a apontar para o WebP final")
	assert.Contains(t, m.MediaUrl, ".webp")
	var d map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(m.DataJson), &d))
	assert.Equal(t, "image/webp", d["mimetype"])
	assert.Equal(t, true, d["animated"])
	assert.Equal(t, "a.png", d["fileName"], "os demais dados da mensagem se mantêm")
	assert.Contains(t, e.bc.events, "room:appMessage")
}

// Só figurinha enviada por nós pode ter o arquivo trocado: um evento "replaced" nunca mexe em outra mensagem.
func TestHandleMediaDownloaded_ReplacedNaoTrocaMidiaQueNaoEFigurinhaNossa(t *testing.T) {
	e := stickerSentEnv(t)
	require.NoError(t, e.db.Create(&models.Message{ID: "FOTO1", MediaType: "image", MediaUrl: "/public/media/foto.jpg", FromMe: true, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "STKRECEBIDA", MediaType: "sticker", MediaUrl: "/public/media/dela.webp", FromMe: false, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)

	require.NoError(t, e.send("FOTO1", nil))
	require.NoError(t, e.send("STKRECEBIDA", nil))

	assert.Equal(t, "/public/media/foto.jpg", e.get("FOTO1").MediaUrl)
	assert.Equal(t, "/public/media/dela.webp", e.get("STKRECEBIDA").MediaUrl)
}

func TestHandleMediaDownloaded_ReplacedSemMensagemNaoQuebra(t *testing.T) {
	e := stickerSentEnv(t)
	require.NoError(t, e.send("NAO_EXISTE", nil))
}

// O GIF enviado vira vídeo: a mensagem passa a "video" com a marca gif (e gifTrimmed quando foi cortado).
func TestHandleMediaDownloaded_GIFEnviadoViraVideoComMarcaGif(t *testing.T) {
	e := stickerSentEnv(t)
	require.NoError(t, e.db.Create(&models.Message{ID: "GIF1", MediaType: "image", MediaUrl: "/public/media/orig.gif", FromMe: true,
		TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{"fileName":"a.gif","mimetype":"image/gif"}`}).Error)

	require.NoError(t, e.send("GIF1", map[string]interface{}{"gif": true, "trimmed": true, "mimetype": "video/mp4", "mediaType": "video"}))

	m := e.get("GIF1")
	assert.Equal(t, "video", m.MediaType, "o GIF passa a ser um vídeo")
	assert.Contains(t, m.MediaUrl, ".mp4")
	var d map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(m.DataJson), &d))
	assert.Equal(t, true, d["gif"])
	assert.Equal(t, true, d["gifTrimmed"])
	assert.Equal(t, "video/mp4", d["mimetype"])
	assert.Equal(t, "a.gif", d["fileName"])
}

func TestHandleMediaDownloaded_GIFSemCorteNaoMarcaGifTrimmed(t *testing.T) {
	e := stickerSentEnv(t)
	require.NoError(t, e.db.Create(&models.Message{ID: "GIF2", MediaType: "image", MediaUrl: "/public/media/o.gif", FromMe: true, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)
	require.NoError(t, e.send("GIF2", map[string]interface{}{"gif": true, "mimetype": "video/mp4"}))
	var d map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(e.get("GIF2").DataJson), &d))
	assert.Equal(t, true, d["gif"])
	assert.NotContains(t, d, "gifTrimmed")
}

// gif=true só troca imagem/vídeo nossos; nunca figurinha, documento ou mensagem recebida.
func TestHandleMediaDownloaded_GIFNaoTrocaOQueNaoEImagemNossa(t *testing.T) {
	e := stickerSentEnv(t)
	require.NoError(t, e.db.Create(&models.Message{ID: "DOC1", MediaType: "document", MediaUrl: "/public/media/d.pdf", FromMe: true, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "RECEB1", MediaType: "image", MediaUrl: "/public/media/r.jpg", FromMe: false, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "STK9", MediaType: "sticker", MediaUrl: "/public/media/s.webp", FromMe: true, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)

	for _, id := range []string{"DOC1", "RECEB1", "STK9"} {
		require.NoError(t, e.send(id, map[string]interface{}{"gif": true, "mimetype": "video/mp4"}))
	}
	assert.Equal(t, "/public/media/d.pdf", e.get("DOC1").MediaUrl)
	assert.Equal(t, "/public/media/r.jpg", e.get("RECEB1").MediaUrl)
	assert.Equal(t, "/public/media/s.webp", e.get("STK9").MediaUrl)
}
