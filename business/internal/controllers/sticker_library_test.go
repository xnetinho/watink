package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stickerEnv struct {
	db     *gorm.DB
	tenant uuid.UUID
	tk     models.Ticket
	router func(tenant uuid.UUID) *gin.Engine
}

func newStickerEnv(t *testing.T) stickerEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })
	require.NoError(t, os.MkdirAll(filepath.Join("public", "media"), 0o755))

	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)

	ctrl := NewStickerLibraryController()
	router := func(tid uuid.UUID) *gin.Engine {
		r := gin.New()
		r.Use(testScopedMiddleware(db, tid.String()))
		r.GET("/stickers", ctrl.List)
		r.POST("/stickers", ctrl.Save)
		r.DELETE("/stickers/:stickerId", ctrl.Delete)
		return r
	}
	return stickerEnv{db: db, tenant: tenant, tk: tk, router: router}
}

// file grava um arquivo no armazenamento de mídia e devolve a URL, como o mediastore faz.
func (e stickerEnv) file(t *testing.T, name, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join("public", "media", name), []byte(content), 0o644))
	return "/public/media/" + name
}

func (e stickerEnv) msg(t *testing.T, id, mediaType, url, data string) {
	t.Helper()
	require.NoError(t, e.db.Create(&models.Message{ID: id, MediaType: mediaType, MediaUrl: url, DataJson: data, TicketID: e.tk.ID, TenantID: e.tenant}).Error)
}

func stickerCall(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestStickerLibrary_SalvaUmaFigurinhaRecebidaOuEnviada(t *testing.T) {
	e := newStickerEnv(t)
	e.msg(t, "RECEBIDA", "sticker", e.file(t, "a.webp", "WEBP-A"), `{"mimetype":"image/webp"}`)
	e.msg(t, "ENVIADA", "sticker", e.file(t, "b.webp", "WEBP-B"), `{"mimetype":"image/webp","animated":true}`)
	r := e.router(e.tenant)

	rec := stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"RECEBIDA"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var a models.StickerLibraryItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
	assert.Equal(t, "/public/media/a.webp", a.MediaUrl)
	assert.False(t, a.Animated)
	assert.NotEmpty(t, a.Hash)
	require.NotNil(t, a.SavedBy, "guarda quem salvou")

	rec = stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"ENVIADA"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var b models.StickerLibraryItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &b))
	assert.True(t, b.Animated, "a marca de animada vem da mensagem")
	assert.NotEqual(t, a.ID, b.ID)
}

func TestStickerLibrary_MesmoConteudoNaoDuplicaMesmoEmMensagensDiferentes(t *testing.T) {
	e := newStickerEnv(t)
	// duas mensagens, dois arquivos com NOMES diferentes mas o MESMO conteúdo: é a mesma figurinha
	e.msg(t, "M1", "sticker", e.file(t, "um.webp", "MESMO-CONTEUDO"), `{}`)
	e.msg(t, "M2", "sticker", e.file(t, "dois.webp", "MESMO-CONTEUDO"), `{}`)
	r := e.router(e.tenant)

	var first, second models.StickerLibraryItem
	rec := stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"M1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &first))
	rec = stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"M2"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &second))

	assert.Equal(t, first.ID, second.ID, "salvar de novo devolve a que já existe")
	var n int64
	require.NoError(t, e.db.Model(&models.StickerLibraryItem{}).Where(`"tenantId" = ?`, e.tenant).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}

func TestStickerLibrary_RecusaOQueNaoEFigurinha(t *testing.T) {
	e := newStickerEnv(t)
	e.msg(t, "FOTO", "image", e.file(t, "f.jpg", "JPG"), `{}`)
	e.msg(t, "SEM_ARQUIVO", "sticker", "", `{}`)
	e.msg(t, "EXTERNA", "sticker", "https://cdn.exemplo.com/x.webp", `{}`)
	e.msg(t, "SUMIDA", "sticker", "/public/media/nao-existe.webp", `{}`)
	require.NoError(t, e.db.Create(&models.Message{ID: "APAGADA", MediaType: "sticker", MediaUrl: e.file(t, "ap.webp", "AP"), IsDeleted: true, TicketID: e.tk.ID, TenantID: e.tenant, DataJson: `{}`}).Error)
	r := e.router(e.tenant)

	for _, id := range []string{"FOTO", "SEM_ARQUIVO", "EXTERNA", "SUMIDA", "APAGADA"} {
		rec := stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"`+id+`"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, id)
	}
	assert.Equal(t, http.StatusNotFound, stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"NAO_EXISTE"}`).Code)
	assert.Equal(t, http.StatusBadRequest, stickerCall(t, r, http.MethodPost, "/stickers", `{}`).Code)
	assert.Equal(t, http.StatusBadRequest, stickerCall(t, r, http.MethodPost, "/stickers", `não é json`).Code)

	var n int64
	require.NoError(t, e.db.Model(&models.StickerLibraryItem{}).Count(&n).Error)
	assert.Zero(t, n, "nada recusado pode ter sido gravado")
}

// A biblioteca é da empresa: outra empresa nunca vê, salva a partir de mensagem alheia nem remove a figurinha de outra.
func TestStickerLibrary_IsolaUmaEmpresaDaOutra(t *testing.T) {
	e := newStickerEnv(t)
	e.msg(t, "M1", "sticker", e.file(t, "a.webp", "A"), `{}`)
	rec := stickerCall(t, e.router(e.tenant), http.MethodPost, "/stickers", `{"messageId":"M1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var mine models.StickerLibraryItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mine))

	other := uuid.New()
	require.NoError(t, e.db.Create(&models.Tenant{ID: other, Name: "Outra"}).Error)
	ro := e.router(other)

	rec = stickerCall(t, ro, http.MethodGet, "/stickers", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var seen []models.StickerLibraryItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &seen))
	assert.Empty(t, seen, "a outra empresa não enxerga a biblioteca")

	assert.Equal(t, http.StatusNotFound, stickerCall(t, ro, http.MethodPost, "/stickers", `{"messageId":"M1"}`).Code, "mensagem de outra empresa não pode ser salva")
	assert.Equal(t, http.StatusNotFound, stickerCall(t, ro, http.MethodDelete, "/stickers/"+itoa(mine.ID), "").Code, "nem removida")

	var n int64
	require.NoError(t, e.db.Model(&models.StickerLibraryItem{}).Where(`"tenantId" = ?`, e.tenant).Count(&n).Error)
	assert.EqualValues(t, 1, n, "a figurinha da empresa dona continua lá")
}

// O mesmo arquivo pode estar na biblioteca de duas empresas: o índice único é por empresa e conteúdo.
func TestStickerLibrary_MesmaFigurinhaEmEmpresasDiferentesNaoColide(t *testing.T) {
	e := newStickerEnv(t)
	e.msg(t, "M1", "sticker", e.file(t, "a.webp", "IGUAL"), `{}`)
	require.Equal(t, http.StatusOK, stickerCall(t, e.router(e.tenant), http.MethodPost, "/stickers", `{"messageId":"M1"}`).Code)

	other := uuid.New()
	require.NoError(t, e.db.Create(&models.Tenant{ID: other, Name: "Outra"}).Error)
	wa2 := models.Whatsapp{Name: "W2", TenantID: other, Status: "CONNECTED"}
	require.NoError(t, e.db.Create(&wa2).Error)
	ct2 := models.Contact{Name: "C2", Number: "5511999990002", TenantID: other}
	require.NoError(t, e.db.Create(&ct2).Error)
	tk2 := models.Ticket{ContactID: ct2.ID, WhatsappID: wa2.ID, TenantID: other, Status: "open"}
	require.NoError(t, e.db.Create(&tk2).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "M2", MediaType: "sticker", MediaUrl: "/public/media/a.webp", TicketID: tk2.ID, TenantID: other, DataJson: `{}`}).Error)

	assert.Equal(t, http.StatusOK, stickerCall(t, e.router(other), http.MethodPost, "/stickers", `{"messageId":"M2"}`).Code)
	var n int64
	require.NoError(t, e.db.Model(&models.StickerLibraryItem{}).Count(&n).Error)
	assert.EqualValues(t, 2, n, "uma linha por empresa")
}

func TestStickerLibrary_ListaMaisRecentePrimeiroERemove(t *testing.T) {
	e := newStickerEnv(t)
	e.msg(t, "M1", "sticker", e.file(t, "a.webp", "A"), `{}`)
	e.msg(t, "M2", "sticker", e.file(t, "b.webp", "B"), `{}`)
	r := e.router(e.tenant)
	require.Equal(t, http.StatusOK, stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"M1"}`).Code)
	require.Equal(t, http.StatusOK, stickerCall(t, r, http.MethodPost, "/stickers", `{"messageId":"M2"}`).Code)

	var items []models.StickerLibraryItem
	rec := stickerCall(t, r, http.MethodGet, "/stickers", "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
	require.Len(t, items, 2)
	assert.Equal(t, "/public/media/b.webp", items[0].MediaUrl, "a mais recente vem primeiro")

	assert.Equal(t, http.StatusOK, stickerCall(t, r, http.MethodDelete, "/stickers/"+itoa(items[0].ID), "").Code)
	assert.Equal(t, http.StatusNotFound, stickerCall(t, r, http.MethodDelete, "/stickers/"+itoa(items[0].ID), "").Code, "remover de novo é 404")
	assert.Equal(t, http.StatusBadRequest, stickerCall(t, r, http.MethodDelete, "/stickers/abc", "").Code)

	rec = stickerCall(t, r, http.MethodGet, "/stickers", "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
	require.Len(t, items, 1)

	_, statErr := os.Stat(filepath.Join("public", "media", "b.webp"))
	assert.NoError(t, statErr, "o arquivo continua no armazenamento: outras mensagens podem apontar para ele")
}
