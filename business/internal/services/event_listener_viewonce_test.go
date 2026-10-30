package services_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inTempDir roda o teste num diretório temporário: o mediastore grava em public/media relativo ao diretório de trabalho e
// não pode sujar o repositório.
func inTempDir(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(old) })
}

type voBroadcast struct{ events []string }

func (b *voBroadcast) EmitToRoom(_, _, ev string, _ interface{}) {
	b.events = append(b.events, "room:"+ev)
}
func (b *voBroadcast) EmitToTenantRoom(_, ev string, _ interface{}) {
	b.events = append(b.events, "tenant:"+ev)
}
func (b *voBroadcast) EmitToNamespace(_, _ string, _ interface{}) {}

// A visualização única recuperada vira mídia comum: a mensagem de AVISO passa a ser a foto, e a prévia do ticket acompanha.
func TestHandleMediaDownloaded_VisualizacaoUnicaRecuperadaViraMidiaComum(t *testing.T) {
	inTempDir(t)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open", LastMessageType: "view_once"}
	require.NoError(t, db.Create(&tk).Error)
	require.NoError(t, db.Create(&models.Message{
		ID: "AVISO1", MediaType: "view_once", TicketID: tk.ID, TenantID: tenant,
		DataJson: `{"mediaStatus":"unavailable","pushName":"Cliente"}`,
	}).Error)
	// uma mensagem comum, para provar que só o aviso muda
	require.NoError(t, db.Create(&models.Message{ID: "COMUM1", MediaType: "image", TicketID: tk.ID, TenantID: tenant, DataJson: `{}`}).Error)

	bc := &voBroadcast{}
	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(db), nil, repository.NewGORMTicketRepo(db), nil, bc, db, nil, nil, nil, nil)

	jpeg := []byte("\xff\xd8\xff\xe0FOTO-DE-TESTE")
	payload, _ := json.Marshal(map[string]interface{}{
		"messageId": "AVISO1", "mediaData": base64.StdEncoding.EncodeToString(jpeg), "mimetype": "image/jpeg",
		"mediaType": "image", "viewOnce": true, "error": "",
	})
	require.NoError(t, el.HandleMediaDownloadedEvent(context.Background(), payload, tenant))

	var m models.Message
	require.NoError(t, db.Where("id = ?", "AVISO1").Take(&m).Error)
	assert.Equal(t, "image", m.MediaType, "o aviso vira a foto")
	assert.NotEmpty(t, m.MediaUrl, "o arquivo fica guardado no Watink")
	assert.Contains(t, m.MediaUrl, ".jpg")
	var d map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(m.DataJson), &d))
	assert.Equal(t, "downloaded", d["mediaStatus"])
	assert.Equal(t, "image/jpeg", d["mimetype"])
	assert.Equal(t, true, d["recoveredViewOnce"])
	assert.Equal(t, "Cliente", d["pushName"], "os dados do remetente do aviso se mantêm")

	var tkAfter models.Ticket
	require.NoError(t, db.Where("id = ?", tk.ID).Take(&tkAfter).Error)
	assert.Equal(t, "image", tkAfter.LastMessageType, "a prévia deixa de dizer 'Visualização única'")
	assert.Contains(t, bc.events, "room:appMessage", "a tela é avisada")
	assert.Contains(t, bc.events, "tenant:ticket")
}

// Se já chegou outra mensagem depois do aviso, a prévia do ticket não pode voltar a apontar para a foto.
func TestHandleMediaDownloaded_NaoMexeNaPreviaSeAvisoJaNaoEAUltima(t *testing.T) {
	inTempDir(t)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open", LastMessage: "Obrigado!", LastMessageType: ""}
	require.NoError(t, db.Create(&tk).Error)
	require.NoError(t, db.Create(&models.Message{ID: "AVISO2", MediaType: "view_once", TicketID: tk.ID, TenantID: tenant, DataJson: `{}`}).Error)

	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(db), nil, repository.NewGORMTicketRepo(db), nil, &voBroadcast{}, db, nil, nil, nil, nil)
	payload, _ := json.Marshal(map[string]interface{}{
		"messageId": "AVISO2", "mediaData": base64.StdEncoding.EncodeToString([]byte("\xff\xd8\xff\xe0X")), "mimetype": "image/jpeg", "mediaType": "image", "viewOnce": true,
	})
	require.NoError(t, el.HandleMediaDownloadedEvent(context.Background(), payload, tenant))

	var tkAfter models.Ticket
	require.NoError(t, db.Where("id = ?", tk.ID).Take(&tkAfter).Error)
	assert.Equal(t, "Obrigado!", tkAfter.LastMessage, "a última mensagem é outra: a prévia fica como está")
}

// Download comum (botão da tela) não muda o tipo de ninguém, e visualização única só vale sobre o aviso.
func TestHandleMediaDownloaded_ComumNaoTrocaOTipo(t *testing.T) {
	inTempDir(t)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)
	require.NoError(t, db.Create(&models.Message{ID: "IMG1", MediaType: "image", TicketID: tk.ID, TenantID: tenant, DataJson: `{"mediaProto":"x"}`}).Error)
	require.NoError(t, db.Create(&models.Message{ID: "TXT1", MediaType: "chat", Body: "oi", TicketID: tk.ID, TenantID: tenant, DataJson: `{}`}).Error)

	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(db), nil, repository.NewGORMTicketRepo(db), nil, &voBroadcast{}, db, nil, nil, nil, nil)
	jpg := base64.StdEncoding.EncodeToString([]byte("\xff\xd8\xff\xe0X"))

	p1, _ := json.Marshal(map[string]interface{}{"messageId": "IMG1", "mediaData": jpg, "mimetype": "image/jpeg"})
	require.NoError(t, el.HandleMediaDownloadedEvent(context.Background(), p1, tenant))
	// um evento de visualização única apontando para uma mensagem que NÃO é aviso: não pode trocar o tipo dela
	p2, _ := json.Marshal(map[string]interface{}{"messageId": "TXT1", "mediaData": jpg, "mimetype": "image/jpeg", "mediaType": "image", "viewOnce": true})
	require.NoError(t, el.HandleMediaDownloadedEvent(context.Background(), p2, tenant))

	var img, txt models.Message
	require.NoError(t, db.Where("id = ?", "IMG1").Take(&img).Error)
	require.NoError(t, db.Where("id = ?", "TXT1").Take(&txt).Error)
	assert.Equal(t, "image", img.MediaType)
	assert.NotEmpty(t, img.MediaUrl)
	assert.Equal(t, "chat", txt.MediaType, "só o aviso de visualização única pode virar mídia")
}
