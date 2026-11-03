package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func postJSON(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func libTenant(t *testing.T, db *gorm.DB, n string) (uuid.UUID, models.Ticket) {
	t.Helper()
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T" + n}).Error)
	wa := models.Whatsapp{Name: "W" + n, TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C" + n, Number: "55119999900" + n, TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)
	return tenant, tk
}

// Enviar uma figurinha da biblioteca: o comando ao engine leva o arquivo guardado como figurinha, sem legenda, e a mensagem
// gravada aponta para o mesmo arquivo.
func TestSendMessage_FiguraDaBiblioteca(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })

	db := testutil.NewTestDB(t)
	tenantA, tkA := libTenant(t, db, "1")
	tenantB, _ := libTenant(t, db, "2")
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)

	item := models.StickerLibraryItem{TenantID: tenantA, Hash: "h1", MediaUrl: "/public/media/lib.webp"}
	require.NoError(t, db.Create(&item).Error)
	other := models.StickerLibraryItem{TenantID: tenantB, Hash: "h2", MediaUrl: "/public/media/alheia.webp"}
	require.NoError(t, db.Create(&other).Error)
	require.NoError(t, os.MkdirAll(filepath.Join("public", "media"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join("public", "media", "lib.webp"), []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), 0o644))

	rA := gin.New()
	rA.Use(testScopedMiddleware(db, tenantA.String()))
	rA.POST("/messages/:ticketId", mc.SendMessage)

	t.Run("manda a figurinha guardada, sem legenda, como figurinha", func(t *testing.T) {
		pub.calls = nil
		rec := postJSON(t, rA, "/messages/"+strconv.Itoa(tkA.ID), `{"stickerId":`+strconv.Itoa(item.ID)+`,"body":"legenda que não pode sair"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Len(t, pub.calls, 1)
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		assert.Equal(t, "sticker", payload["mediaType"])
		assert.Equal(t, "/public/media/lib.webp", payload["mediaUrl"], "o arquivo é o da biblioteca")
		assert.Equal(t, "image/webp", payload["mimeType"])
		assert.Empty(t, payload["body"])
		var m models.Message
		require.NoError(t, db.Where(`id = ?`, payload["messageId"]).Take(&m).Error)
		assert.Equal(t, "sticker", m.MediaType)
		assert.Equal(t, "/public/media/lib.webp", m.MediaUrl)
		assert.Empty(t, m.Body)
	})

	t.Run("figurinha de OUTRA empresa não pode ser enviada", func(t *testing.T) {
		pub.calls = nil
		rec := postJSON(t, rA, "/messages/"+strconv.Itoa(tkA.ID), `{"stickerId":`+strconv.Itoa(other.ID)+`}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Empty(t, pub.calls, "nada pode ter sido enviado")
	})

	t.Run("id que não existe é 404 e nada sai", func(t *testing.T) {
		pub.calls = nil
		assert.Equal(t, http.StatusNotFound, postJSON(t, rA, "/messages/"+strconv.Itoa(tkA.ID), `{"stickerId":99999}`).Code)
		assert.Empty(t, pub.calls)
	})

	t.Run("a URL do cliente é ignorada quando vem stickerId: nenhum caminho de disco vem de fora", func(t *testing.T) {
		pub.calls = nil
		rec := postJSON(t, rA, "/messages/"+strconv.Itoa(tkA.ID), `{"stickerId":`+strconv.Itoa(item.ID)+`,"mediaUrl":"/public/media/../../etc/passwd","mediaType":"document"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		assert.Equal(t, "/public/media/lib.webp", payload["mediaUrl"])
		assert.Equal(t, "sticker", payload["mediaType"])
	})
}
