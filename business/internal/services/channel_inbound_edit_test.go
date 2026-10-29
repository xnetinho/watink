package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A localização ao vivo do Telegram chega como edited_message a cada ~30s, com o mesmo message_id. Cada edição precisa
// atualizar a mensagem original: não pode duplicar nem ser descartada pelo dedup.
func TestChannelInbound_EdicaoAtualizaAOriginal(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "T Edit"}).Error)
	conn := models.ChannelConnection{TenantID: tenantID, Kind: "telegram", Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)

	ticketRepo, contactRepo, messageRepo := repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db), repository.NewGORMMessageRepo(db)
	uc := usecases.NewReceiveMessageUseCase(nil, messageRepo, contactRepo, ticketRepo, nil, nil, nil)
	svc := services.NewChannelInboundService(db, newMockRedis(), ticketRepo, contactRepo, messageRepo, uc, nil, nil)

	base := domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: conn.ID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: "23", ConversationKey: "chat-1", Sender: domain.InboundSender{ExternalID: "chat-1", Name: "Cli"},
		MediaType: "location", Body: "|https://maps/pos1|", Timestamp: time.Now(),
		Rich: map[string]interface{}{"live": true},
	}
	ctx := context.Background()
	_, err := svc.Receive(ctx, base)
	require.NoError(t, err)

	for _, pos := range []string{"pos2", "pos3"} {
		edit := base
		edit.Edited, edit.Body = true, "|https://maps/"+pos+"|"
		_, err := svc.Receive(ctx, edit)
		require.NoError(t, err)
	}

	var msgs []models.Message
	require.NoError(t, db.Where(`"tenantId" = ?`, tenantID).Find(&msgs).Error)
	require.Len(t, msgs, 1, "edições não podem criar mensagens novas")
	assert.Equal(t, "|https://maps/pos3|", msgs[0].Body, "a última posição vence")
	assert.JSONEq(t, `{"rich":{"live":true}}`, msgs[0].DataJson)
}

func TestChannelInbound_EncaminhadaFicaNoDataJson(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "T Fwd"}).Error)
	conn := models.ChannelConnection{TenantID: tenantID, Kind: "telegram", Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)
	ticketRepo, contactRepo, messageRepo := repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db), repository.NewGORMMessageRepo(db)
	uc := usecases.NewReceiveMessageUseCase(nil, messageRepo, contactRepo, ticketRepo, nil, nil, nil)
	svc := services.NewChannelInboundService(db, newMockRedis(), ticketRepo, contactRepo, messageRepo, uc, nil, nil)

	_, err := svc.Receive(context.Background(), domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: conn.ID, Channel: domain.ChannelKindTelegram, ExternalMessageID: "7",
		ConversationKey: "c", Sender: domain.InboundSender{ExternalID: "c", Name: "X"}, Body: "olha", MediaType: "chat",
		Forwarded: true, Timestamp: time.Now(),
	})
	require.NoError(t, err)
	var m models.Message
	require.NoError(t, db.Where(`"tenantId" = ?`, tenantID).First(&m).Error)
	assert.JSONEq(t, `{"forwarded":true}`, m.DataJson)
}

// Texto editado no Telegram: antes o corpo era sobrescrito e as versões anteriores se perdiam. Agora cada edição entra
// na cadeia (a versão 1 é o texto original) e a localização ao vivo, que não é edição de texto, não polui o histórico.
func TestChannelInbound_TextoEditadoGuardaACadeia(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "T Chain"}).Error)
	conn := models.ChannelConnection{TenantID: tenantID, Kind: "telegram", Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)
	ticketRepo, contactRepo, messageRepo := repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db), repository.NewGORMMessageRepo(db)
	uc := usecases.NewReceiveMessageUseCase(nil, messageRepo, contactRepo, ticketRepo, nil, nil, nil)
	svc := services.NewChannelInboundService(db, newMockRedis(), ticketRepo, contactRepo, messageRepo, uc, nil, nil)

	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	base := domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: conn.ID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: "50", ConversationKey: "chat-9", Sender: domain.InboundSender{ExternalID: "chat-9", Name: "Cli"},
		MediaType: "chat", Body: "Oi", Timestamp: t0,
	}
	ctx := context.Background()
	_, err := svc.Receive(ctx, base)
	require.NoError(t, err)

	for i, body := range []string{"Olá", "Olá, tudo bem?"} {
		edit := base
		edit.Edited, edit.Body, edit.EditedAt = true, body, t0.Add(time.Duration(i+1)*time.Minute)
		_, err := svc.Receive(ctx, edit)
		require.NoError(t, err)
	}
	// o mesmo evento reentregue não duplica
	again := base
	again.Edited, again.Body, again.EditedAt = true, "Olá, tudo bem?", t0.Add(2*time.Minute)
	_, err = svc.Receive(ctx, again)
	require.NoError(t, err)

	var msgs []models.Message
	require.NoError(t, db.Where(`"tenantId" = ?`, tenantID).Find(&msgs).Error)
	require.Len(t, msgs, 1, "edição não cria mensagem nova")
	assert.Equal(t, "Olá, tudo bem?", msgs[0].Body)
	assert.Equal(t, 2, msgs[0].EditCount)

	chain, err := services.MessageEdits(ctx, db, tenantID, msgs[0].ID)
	require.NoError(t, err)
	require.Len(t, chain, 3)
	assert.Equal(t, []string{"Oi", "Olá", "Olá, tudo bem?"}, []string{chain[0].Body, chain[1].Body, chain[2].Body})
}

func TestChannelInbound_LocalizacaoAoVivoNaoCriaVersoes(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "T Loc"}).Error)
	conn := models.ChannelConnection{TenantID: tenantID, Kind: "telegram", Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)
	ticketRepo, contactRepo, messageRepo := repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db), repository.NewGORMMessageRepo(db)
	uc := usecases.NewReceiveMessageUseCase(nil, messageRepo, contactRepo, ticketRepo, nil, nil, nil)
	svc := services.NewChannelInboundService(db, newMockRedis(), ticketRepo, contactRepo, messageRepo, uc, nil, nil)

	base := domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: conn.ID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: "60", ConversationKey: "chat-8", Sender: domain.InboundSender{ExternalID: "chat-8", Name: "Cli"},
		MediaType: "location", Body: "|https://maps/pos1|", Timestamp: time.Now(), Rich: map[string]interface{}{"live": true},
	}
	_, err := svc.Receive(context.Background(), base)
	require.NoError(t, err)
	for _, pos := range []string{"pos2", "pos3", "pos4"} {
		edit := base
		edit.Edited, edit.Body = true, "|https://maps/"+pos+"|"
		_, err := svc.Receive(context.Background(), edit)
		require.NoError(t, err)
	}
	var n int64
	require.NoError(t, db.Model(&models.MessageEdit{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	assert.Zero(t, n, "a posição ao vivo é atualização de cartão: não gera versões")
}
