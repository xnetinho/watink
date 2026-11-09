package repository

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedContactAndTenant(t *testing.T) (*GORMTicketRepository, uuid.UUID, int) {
	t.Helper()
	db := setupTicketTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&TenantTest{ID: tenant, Name: "T Conversa"}).Error)
	contact := &models.Contact{Name: "Ana", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(contact).Error)
	return NewGORMTicketRepo(db), tenant, contact.ID
}

// O ticket pendente é achado pela CHAVE DA CONVERSA, a mesma do índice único idx_tickets_open_conversation, e não pelo contato. Antes o filtro era
// (contato, conexão, pendente) e ignorava a chave: no grupo do Telegram o contato é quem escreve e a conversa é o grupo, então a mensagem do grupo caía no
// ticket PRIVADO da pessoa (visto no lt em 9/nov/2026). No Mercado Livre e na OLX o mesmo contato tem uma conversa por pedido/anúncio e elas seriam fundidas.
func TestFindOrCreatePending_MesmoContatoEmDuasConversasSaoDoisTickets(t *testing.T) {
	repo, tenant, contactID := seedContactAndTenant(t)
	ctx := context.Background()
	mk := func(key string) *domain.Ticket {
		return &domain.Ticket{Status: "pending", ContactID: contactID, ChannelConnectionID: 2, TenantID: tenant, ConversationKey: key}
	}

	privado, err := repo.FindOrCreatePending(ctx, mk(domain.TelegramConversationKey(555)))
	require.NoError(t, err)
	grupo, err := repo.FindOrCreatePending(ctx, mk(domain.TelegramConversationKey(-1001234567890)))
	require.NoError(t, err)

	assert.NotEqual(t, privado.ID, grupo.ID, "o grupo não pode cair no ticket privado do mesmo remetente")
	assert.Equal(t, "tg:555", privado.ConversationKey)
	assert.Equal(t, "tg:-1001234567890", grupo.ConversationKey)
}

func TestFindOrCreatePending_MesmaChaveDevolveOMesmoTicket(t *testing.T) {
	repo, tenant, contactID := seedContactAndTenant(t)
	ctx := context.Background()
	in := func() *domain.Ticket {
		return &domain.Ticket{Status: "pending", ContactID: contactID, ChannelConnectionID: 2, TenantID: tenant, ConversationKey: "tg:555"}
	}
	a, err := repo.FindOrCreatePending(ctx, in())
	require.NoError(t, err)
	b, err := repo.FindOrCreatePending(ctx, in())
	require.NoError(t, err)
	assert.Equal(t, a.ID, b.ID)
}

// A chave só vale dentro da conexão: o mesmo chat_id em dois bots são duas conversas.
func TestFindOrCreatePending_MesmaChaveEmOutraConexaoEOutroTicket(t *testing.T) {
	repo, tenant, contactID := seedContactAndTenant(t)
	ctx := context.Background()
	mk := func(conn int) *domain.Ticket {
		return &domain.Ticket{Status: "pending", ContactID: contactID, ChannelConnectionID: conn, TenantID: tenant, ConversationKey: "tg:555"}
	}
	a, err := repo.FindOrCreatePending(ctx, mk(2))
	require.NoError(t, err)
	b, err := repo.FindOrCreatePending(ctx, mk(3))
	require.NoError(t, err)
	assert.NotEqual(t, a.ID, b.ID)
}

// Uma chave que já tem ticket ABERTO (não pendente) não gera um segundo: o índice único cobre open e pending, e a busca também.
func TestFindOrCreatePending_ConversaJaAbertaDevolveOAberto(t *testing.T) {
	repo, tenant, contactID := seedContactAndTenant(t)
	ctx := context.Background()
	aberto, err := repo.FindOrCreatePending(ctx, &domain.Ticket{Status: "open", ContactID: contactID, ChannelConnectionID: 2, TenantID: tenant, ConversationKey: "ml:9"})
	require.NoError(t, err)
	again, err := repo.FindOrCreatePending(ctx, &domain.Ticket{Status: "pending", ContactID: contactID, ChannelConnectionID: 2, TenantID: tenant, ConversationKey: "ml:9"})
	require.NoError(t, err)
	assert.Equal(t, aberto.ID, again.ID, "não pode abrir um segundo ticket para a mesma conversa")
}

// Sem chave de conversa a busca voltaria ao critério frágil (por contato): é recusada em vez de arriscar fundir conversas.
func TestFindOrCreatePending_SemChaveDeConversaERecusado(t *testing.T) {
	repo, tenant, contactID := seedContactAndTenant(t)
	_, err := repo.FindOrCreatePending(context.Background(), &domain.Ticket{Status: "pending", ContactID: contactID, ChannelConnectionID: 2, TenantID: tenant})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrConversationKeyRequired)
}

// Tenant diferente nunca enxerga o ticket do outro, mesmo com a mesma chave e conexão.
func TestFindOrCreatePending_IsolaPorEmpresa(t *testing.T) {
	db := setupTicketTestDB(t)
	a, b := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&TenantTest{ID: a, Name: "A"}).Error)
	require.NoError(t, db.Create(&TenantTest{ID: b, Name: "B"}).Error)
	ca := &models.Contact{Name: "x", Number: "551", TenantID: a}
	cb := &models.Contact{Name: "y", Number: "552", TenantID: b}
	require.NoError(t, db.Create(ca).Error)
	require.NoError(t, db.Create(cb).Error)
	repo := NewGORMTicketRepo(db)
	ta, err := repo.FindOrCreatePending(context.Background(), &domain.Ticket{Status: "pending", ContactID: ca.ID, ChannelConnectionID: 2, TenantID: a, ConversationKey: "tg:555"})
	require.NoError(t, err)
	tb, err := repo.FindOrCreatePending(context.Background(), &domain.Ticket{Status: "pending", ContactID: cb.ID, ChannelConnectionID: 2, TenantID: b, ConversationKey: "tg:555"})
	require.NoError(t, err)
	assert.NotEqual(t, ta.ID, tb.ID)
	assert.Equal(t, b, tb.TenantID)
}
