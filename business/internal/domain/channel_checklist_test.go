package domain_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/channels/fake"
	"github.com/alltomatos/watinkdev/business/internal/channels/telegram"
	"github.com/alltomatos/watinkdev/business/internal/channels/whatsapp"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChannelChecklistSuite valida o checklist executável obrigatório para todo novo canal (Tarefa 9.4).
// Qualquer canal adicionado à plataforma deve satisfazer estas garantias contratuais.
func verifyChannelChecklist(t *testing.T, ch domain.Channel, dummyConn *models.ChannelConnection, dummyContact *models.Contact) {
	t.Helper()
	ctx := context.Background()

	// 1. Kind não pode ser vazio
	kind := ch.Kind()
	assert.NotEmpty(t, string(kind), "o canal deve declarar um Kind não vazio")

	// 2. Capabilities deve ser declarada com limites válidos
	caps := ch.Capabilities()
	assert.Greater(t, caps.MaxTextBytes, 0, "o canal deve definir MaxTextBytes positivo")

	// 3. ResolveAddress deve resolver um endereço não vazio para um contato válido
	addr, err := ch.ResolveAddress(ctx, dummyConn, dummyContact)
	require.NoError(t, err, "ResolveAddress não deve falhar para contato válido")
	assert.NotEmpty(t, addr, "ResolveAddress deve devolver um endereço válido")

	// 4. Disconnect não deve gerar pânico
	require.NoError(t, ch.Disconnect(ctx, dummyConn), "Disconnect deve executar sem erro")

	// 5. ParseInbound não deve gerar pânico com payload vazio
	_, _ = ch.ParseInbound(ctx, dummyConn, []byte("{}"), nil)
}

func TestNewChannelChecklist_FakeChannel(t *testing.T) {
	fakeCh := fake.NewChannel()
	conn := &models.ChannelConnection{ID: 1, Kind: string(fake.ChannelKindFake), Status: "CONNECTED"}
	contact := &models.Contact{Number: "fake-1234"}

	verifyChannelChecklist(t, fakeCh, conn, contact)

	// Validação de envio e entrega do canal fake
	res, err := fakeCh.Send(context.Background(), conn, domain.OutboundMessage{
		ExternalID: "fake-1234",
		Body:       "Mensagem de teste de conformidade",
	})
	require.NoError(t, err)
	assert.Equal(t, "sent", res.Status)
	assert.NotEmpty(t, res.ExternalMessageID)
}

func TestNewChannelChecklist_WhatsAppChannel(t *testing.T) {
	waCh := whatsapp.NewAdapter(nil)
	conn := &models.ChannelConnection{ID: 2, Kind: string(domain.ChannelKindWhatsApp), Status: "CONNECTED"}
	contact := &models.Contact{Number: "5511999990000"}

	verifyChannelChecklist(t, waCh, conn, contact)
}

func TestNewChannelChecklist_TelegramChannel(t *testing.T) {
	tgCh := telegram.NewAdapter(nil)
	conn := &models.ChannelConnection{ID: 3, Kind: string(domain.ChannelKindTelegram), Status: "CONNECTED"}
	contact := &models.Contact{Number: "tg-chat-999"}

	verifyChannelChecklist(t, tgCh, conn, contact)
}
