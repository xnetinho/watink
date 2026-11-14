package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alltomatos/watinkdev/business/internal/routing"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
)

// Ponto único da fila do ticket novo (PR 2a), contra banco migrado + RLS no papel do app.

func (e *qEnv) setDefaultQueue(t *testing.T, queueID *int) {
	t.Helper()
	_, err := e.admin.Exec(`UPDATE "Tenants" SET "defaultQueueId" = $2 WHERE id = $1`, e.tenant, queueID)
	require.NoError(t, err)
}

func (e *qEnv) linkQueue(t *testing.T, connID, queueID int) {
	t.Helper()
	_, err := e.admin.Exec(`INSERT INTO channel_connection_queues (channel_connection_id, queue_id) VALUES ($1,$2)`, connID, queueID)
	require.NoError(t, err)
}

func (e *qEnv) resolved(t *testing.T, connID int) *int {
	t.Helper()
	var got *int
	err := tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		got = routing.ResolveNewTicketQueue(ctx, e.gdb, e.tenant, connID)
		return nil
	})
	require.NoError(t, err)
	return got
}

func intPtr(v int) *int { return &v }

// A fila da conexão manda quando é a única — é o caso do WhatsApp já configurado.
func TestResolveNewTicketQueue_ConexaoComUmaFila(t *testing.T) {
	e := newQEnv(t)
	q := e.queue(t, "Atendimento")
	conn := e.conn(t, "whatsapp")

	e.linkQueue(t, conn, q)
	// A padrão é outra: quem manda é a fila única da conexão.
	e.setDefaultQueue(t, intPtr(e.queue(t, "Outra")))

	got := e.resolved(t, conn)
	require.NotNil(t, got)
	assert.Equal(t, q, *got)
}

// Sem fila na conexão (Telegram/webchat recém-criado), o ticket herda a padrão.
func TestResolveNewTicketQueue_ConexaoSemFilaUsaPadrao(t *testing.T) {
	e := newQEnv(t)
	defaultQueue := e.queue(t, "Padrão")
	e.setDefaultQueue(t, intPtr(defaultQueue))
	conn := e.conn(t, "telegram")

	got := e.resolved(t, conn)
	require.NotNil(t, got)
	assert.Equal(t, defaultQueue, *got)
}

// Duas filas na mesma conexão é ambíguo: não adivinha, usa a padrão.
func TestResolveNewTicketQueue_ConexaoComDuasFilasUsaPadrao(t *testing.T) {
	e := newQEnv(t)
	q1 := e.queue(t, "Comercial")
	q2 := e.queue(t, "Suporte")
	defaultQueue := e.queue(t, "Padrão")
	e.setDefaultQueue(t, intPtr(defaultQueue))
	conn := e.conn(t, "whatsapp")
	e.linkQueue(t, conn, q1)
	e.linkQueue(t, conn, q2)

	got := e.resolved(t, conn)
	require.NotNil(t, got)
	assert.Equal(t, defaultQueue, *got)
}

// Sem fila na conexão e sem padrão o ticket fica sem fila (só admin vê) — nunca herda fila de outra empresa.
func TestResolveNewTicketQueue_SemFilaESemPadrao(t *testing.T) {
	e := newQEnv(t)
	conn := e.conn(t, "whatsapp")
	assert.Nil(t, e.resolved(t, conn))
}

func TestResolveNewTicketQueue_PadraoDeOutraEmpresaNaoVaza(t *testing.T) {
	e := newQEnv(t)
	conn := e.conn(t, "whatsapp")

	otherTenant := uuid.New()
	_, err := e.admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'B','active',now(),now())`, otherTenant)
	require.NoError(t, err)
	foreignQueue := e.one(t, `INSERT INTO "Queues" (name, color, "tenantId") VALUES ('Alheia','#000',$1) RETURNING id`, otherTenant)
	e.setDefaultQueue(t, intPtr(foreignQueue))

	assert.Nil(t, e.resolved(t, conn), "fila padrão de outra empresa não pode ser herdada")
}

// Fila padrão excluída (dado antigo) não pode aparecer no ticket novo.
func TestResolveNewTicketQueue_PadraoInexistente(t *testing.T) {
	e := newQEnv(t)
	conn := e.conn(t, "whatsapp")
	e.setDefaultQueue(t, intPtr(999999))

	assert.Nil(t, e.resolved(t, conn))
}
