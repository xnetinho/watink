package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	grpChat = "tg:-1001234567890"
	grpID   = "-1001234567890"
)

func tgGroupMsg(tenantID uuid.UUID, connID int, extID, personID, personName, title, body string) domain.InboundMessage {
	return domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: connID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: extID, ConversationKey: grpChat,
		Sender: domain.InboundSender{ExternalID: personID, Name: personName, IsGroup: true, GroupJID: grpID, GroupName: title},
		Body:   body, MediaType: "chat", Timestamp: time.Now(),
	}
}

func tgPrivateMsg(tenantID uuid.UUID, connID int, extID, personID, personName, body string) domain.InboundMessage {
	return domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: connID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: extID, ConversationKey: "tg:" + personID,
		Sender: domain.InboundSender{ExternalID: personID, Name: personName},
		Body:   body, MediaType: "chat", Timestamp: time.Now(),
	}
}

// Em grupo o contato é o GRUPO (como no WhatsApp), não quem escreveu: a lista mostra o nome do grupo e o ticket é um só para o grupo todo.
// Quem escreveu fica na mensagem (participant e pushName), que é o que o balão mostra.
func TestChannelInbound_GrupoTelegram_ContatoEOGrupoERemetenteFicaNaMensagem(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	ctx := context.Background()

	r1, err := svc.Receive(ctx, tgGroupMsg(tenantID, conn.ID, "1", "111", "Ana", "Time de Vendas", "oi pessoal"))
	require.NoError(t, err)
	r2, err := svc.Receive(ctx, tgGroupMsg(tenantID, conn.ID, "2", "222", "Bruno", "Time de Vendas", "olá"))
	require.NoError(t, err)

	assert.Equal(t, r1.Contact.ID, r2.Contact.ID, "duas pessoas no mesmo grupo: um só contato, o grupo")
	assert.Equal(t, r1.Ticket.ID, r2.Ticket.ID, "e um só ticket")
	assert.Equal(t, "Time de Vendas", r1.Contact.Name)
	assert.True(t, r1.Contact.IsGroup)
	assert.True(t, r1.Ticket.IsGroup)

	byExt := func(ext string) models.Message {
		var m models.Message
		require.NoError(t, db.Where(`"ticketId" = ? AND "externalId" = ?`, r1.Ticket.ID, ext).First(&m).Error)
		return m
	}
	ana, bruno := byExt("1"), byExt("2")
	assert.Equal(t, "111", ana.Participant)
	assert.Equal(t, "222", bruno.Participant)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(ana.DataJson), &data))
	assert.Equal(t, "Ana", data["pushName"], "o balão mostra quem escreveu")
	assert.Equal(t, true, data["isGroup"])
}

// A mesma pessoa no privado e no grupo continua sendo duas conversas e dois contatos: o do grupo não rouba o dela.
func TestChannelInbound_GrupoTelegram_PrivadoDaMesmaPessoaNaoSeMistura(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	ctx := context.Background()

	priv, err := svc.Receive(ctx, tgPrivateMsg(tenantID, conn.ID, "1", "111", "Ana", "oi no privado"))
	require.NoError(t, err)
	grp, err := svc.Receive(ctx, tgGroupMsg(tenantID, conn.ID, "2", "111", "Ana", "Time de Vendas", "oi no grupo"))
	require.NoError(t, err)

	assert.NotEqual(t, priv.Contact.ID, grp.Contact.ID)
	assert.NotEqual(t, priv.Ticket.ID, grp.Ticket.ID)
	assert.Equal(t, "Ana", priv.Contact.Name)
	assert.False(t, priv.Contact.IsGroup)
	assert.False(t, priv.Ticket.IsGroup)

	var privMsgs int64
	db.Model(&models.Message{}).Where(`"ticketId" = ?`, priv.Ticket.ID).Count(&privMsgs)
	assert.EqualValues(t, 1, privMsgs, "o ticket privado só tem a mensagem privada")
}

// O título do grupo muda; o contato acompanha (a identidade só guarda o nome de quando foi criada).
func TestChannelInbound_GrupoTelegram_TituloNovoAtualizaOContato(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	ctx := context.Background()

	_, err := svc.Receive(ctx, tgGroupMsg(tenantID, conn.ID, "1", "111", "Ana", "Nome Antigo", "a"))
	require.NoError(t, err)
	r, err := svc.Receive(ctx, tgGroupMsg(tenantID, conn.ID, "2", "111", "Ana", "Nome Novo", "b"))
	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", r.Contact.Name)

	var c models.Contact
	require.NoError(t, db.First(&c, r.Contact.ID).Error)
	assert.Equal(t, "Nome Novo", c.Name)
	assert.True(t, c.IsGroup)
}

// No privado nada muda: sem participant, e o contato é a pessoa.
func TestChannelInbound_Privado_NaoGravaParticipant(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	r, err := svc.Receive(context.Background(), tgPrivateMsg(tenantID, conn.ID, "1", "111", "Ana", "oi"))
	require.NoError(t, err)
	var m models.Message
	require.NoError(t, db.Where(`"ticketId" = ?`, r.Ticket.ID).First(&m).Error)
	assert.Empty(t, m.Participant)
	assert.Equal(t, "{}", m.DataJson)
}
