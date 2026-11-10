package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
)

// Todos os testes abaixo rodam contra o banco migrado de verdade: com as chaves estrangeiras e o RLS reais, no papel do app (sem BYPASSRLS).

type qEnv struct {
	admin  *sql.DB
	gdb    *gorm.DB
	tenant uuid.UUID
}

func newQEnv(t *testing.T) *qEnv {
	t.Helper()
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	return &qEnv{admin: admin, gdb: gdb, tenant: tenant}
}

func (e *qEnv) one(t *testing.T, query string, args ...any) int {
	t.Helper()
	var id int
	require.NoError(t, e.admin.QueryRow(query, args...).Scan(&id))
	return id
}

func (e *qEnv) queue(t *testing.T, name string) int {
	return e.one(t, `INSERT INTO "Queues" (name, color, "tenantId") VALUES ($1,'#000',$2) RETURNING id`, name, e.tenant)
}

func (e *qEnv) user(t *testing.T, email string) int {
	return e.one(t, `INSERT INTO "Users" (name, email, "passwordHash", "tenantId") VALUES ($1,$1,'h',$2) RETURNING id`, email, e.tenant)
}

func (e *qEnv) conn(t *testing.T, name string) int {
	id, err := seedWhatsapp(e.admin, e.tenant, name)
	require.NoError(t, err)
	return id
}

func (e *qEnv) ticket(t *testing.T, key string, connID int, queueID, userID *int, status string, group bool) int {
	t.Helper()
	contact := e.one(t, `INSERT INTO "Contacts" (name, number, "tenantId", "connectionId") VALUES ($1,$1,$2,$3) RETURNING id`, key, e.tenant, connID)
	return e.one(t, `INSERT INTO "Tickets" (status, "contactId", "channelConnectionId", "conversationKey", "tenantId", "queueId", "userId", "isGroup")
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, status, contact, connID, key, e.tenant, queueID, userID, group)
}

func (e *qEnv) count(t *testing.T, query string, args ...any) int { return e.one(t, query, args...) }

func (e *qEnv) deleteQueue(queueID int, moveTo *int) error {
	return tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		return services.DeleteQueue(ctx, e.gdb, e.tenant, queueID, moveTo)
	})
}

func (e *qEnv) deleteUser(userID int, walletTo *int) error {
	return tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		return services.DeleteUser(ctx, e.gdb, e.tenant, userID, walletTo)
	})
}

// Fila sem nada pendurado sai direto. Era o caso em que o banco deixava o ticket órfão: aqui não há ticket, então nada fica apontando.
func TestDeleteQueue_SemDependentes_Exclui(t *testing.T) {
	e := newQEnv(t)
	q := e.queue(t, "vazia")
	require.NoError(t, e.deleteQueue(q, nil))
	assert.Zero(t, e.count(t, `SELECT count(*) FROM "Queues" WHERE id = $1`, q))
}

// Com ticket, usuário, conexão, setor e fila filha, sem destino a exclusão é recusada e NADA muda.
func TestDeleteQueue_ComDependentes_SemDestinoRecusaENaoMudaNada(t *testing.T) {
	e := newQEnv(t)
	q, u, c := e.queue(t, "origem"), e.user(t, "u1@x"), e.conn(t, "A")
	e.ticket(t, "wa:1", c, &q, nil, "pending", false)
	_, err := e.admin.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES ($1,$2)`, u, q)
	require.NoError(t, err)

	err = e.deleteQueue(q, nil)
	var dep *services.QueueHasDependentsError
	require.ErrorAs(t, err, &dep)
	assert.EqualValues(t, 1, dep.Dependents.Tickets)
	assert.EqualValues(t, 1, dep.Dependents.Users)

	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Queues" WHERE id = $1`, q), "a fila continua")
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Tickets" WHERE "queueId" = $1`, q))
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM user_queues WHERE queue_id = $1`, q))
}

// Com destino, tudo passa para a fila nova e a antiga sai sem deixar ticket órfão nem vínculo pendurado.
func TestDeleteQueue_ComDestino_MoveTudo(t *testing.T) {
	e := newQEnv(t)
	from, to, u, c := e.queue(t, "origem"), e.queue(t, "destino"), e.user(t, "u1@x"), e.conn(t, "A")
	tk := e.ticket(t, "wa:1", c, &from, nil, "pending", false)
	_, err := e.admin.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES ($1,$2)`, u, from)
	require.NoError(t, err)
	_, err = e.admin.Exec(`INSERT INTO channel_connection_queues (channel_connection_id, queue_id) VALUES ($1,$2)`, c, from)
	require.NoError(t, err)
	child := e.queue(t, "filha")
	_, err = e.admin.Exec(`UPDATE "Queues" SET "parentId" = $1 WHERE id = $2`, from, child)
	require.NoError(t, err)

	require.NoError(t, e.deleteQueue(from, &to))

	assert.Zero(t, e.count(t, `SELECT count(*) FROM "Queues" WHERE id = $1`, from))
	assert.Equal(t, to, e.count(t, `SELECT "queueId" FROM "Tickets" WHERE id = $1`, tk))
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM user_queues WHERE user_id = $1 AND queue_id = $2`, u, to))
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM channel_connection_queues WHERE channel_connection_id = $1 AND queue_id = $2`, c, to))
	assert.Zero(t, e.count(t, `SELECT count(*) FROM "Tickets" t WHERE t."queueId" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "Queues" q WHERE q.id = t."queueId")`), "nenhum ticket órfão")
	assert.Equal(t, 0, e.count(t, `SELECT count(*) FROM "Queues" WHERE id = $1 AND "parentId" IS NOT NULL`, child), "a filha ficou sem pai")
}

// Quem já está na fila de destino não duplica o vínculo (a chave primária é (user_id, queue_id)).
func TestDeleteQueue_ComDestino_NaoDuplicaVinculoExistente(t *testing.T) {
	e := newQEnv(t)
	from, to, u := e.queue(t, "origem"), e.queue(t, "destino"), e.user(t, "u1@x")
	for _, q := range []int{from, to} {
		_, err := e.admin.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES ($1,$2)`, u, q)
		require.NoError(t, err)
	}
	require.NoError(t, e.deleteQueue(from, &to))
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM user_queues WHERE user_id = $1`, u))
}

func TestDeleteQueue_DestinoInvalido(t *testing.T) {
	e := newQEnv(t)
	from, c := e.queue(t, "origem"), e.conn(t, "A")
	e.ticket(t, "wa:1", c, &from, nil, "pending", false)

	self := from
	assert.ErrorIs(t, e.deleteQueue(from, &self), services.ErrQueueMoveInvalid, "a própria fila")
	none := 999999
	assert.ErrorIs(t, e.deleteQueue(from, &none), services.ErrQueueMoveInvalid, "fila que não existe")
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Tickets" WHERE "queueId" = $1`, from), "nada mudou")
}

// Outra empresa nunca é destino nem origem (RLS real, papel sem BYPASSRLS).
func TestDeleteQueue_OutraEmpresa(t *testing.T) {
	e := newQEnv(t)
	other := uuid.New()
	_, err := e.admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'B','active',now(),now())`, other)
	require.NoError(t, err)
	foreign := e.one(t, `INSERT INTO "Queues" (name, color, "tenantId") VALUES ('alheia','#000',$1) RETURNING id`, other)

	mine, c := e.queue(t, "minha"), e.conn(t, "A")
	e.ticket(t, "wa:1", c, &mine, nil, "pending", false)

	assert.ErrorIs(t, e.deleteQueue(mine, &foreign), services.ErrQueueMoveInvalid, "fila de outra empresa não é destino")
	assert.ErrorIs(t, e.deleteQueue(foreign, nil), services.ErrQueueNotFound, "fila de outra empresa nem aparece")
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Queues" WHERE id = $1`, foreign))
}

// Usuário sem nada sai direto.
func TestDeleteUser_SemDependentes_Exclui(t *testing.T) {
	e := newQEnv(t)
	u := e.user(t, "u1@x")
	require.NoError(t, e.deleteUser(u, nil))
	assert.Zero(t, e.count(t, `SELECT count(*) FROM "Users" WHERE id = $1`, u))
}

// O ticket do usuário volta à fila para outro aceitar (pendente, sem dono); o de grupo e o fechado só perdem o dono. Nada fica preso à chave estrangeira.
func TestDeleteUser_TicketsVoltamParaAFila(t *testing.T) {
	e := newQEnv(t)
	q, u, c := e.queue(t, "fila"), e.user(t, "u1@x"), e.conn(t, "A")
	open := e.ticket(t, "wa:1", c, &q, &u, "open", false)
	grp := e.ticket(t, "wa:2", c, &q, &u, "open", true)
	closed := e.ticket(t, "wa:3", c, &q, &u, "closed", false)

	require.NoError(t, e.deleteUser(u, nil))

	assert.Zero(t, e.count(t, `SELECT count(*) FROM "Users" WHERE id = $1`, u))
	var status string
	var userNull, queueKept bool
	require.NoError(t, e.admin.QueryRow(`SELECT status, "userId" IS NULL, "queueId" = $2 FROM "Tickets" WHERE id = $1`, open, q).Scan(&status, &userNull, &queueKept))
	assert.Equal(t, "pending", status, "volta a pendente para ser aceito")
	assert.True(t, userNull)
	assert.True(t, queueKept, "continua na fila dele")
	require.NoError(t, e.admin.QueryRow(`SELECT status, "userId" IS NULL, true FROM "Tickets" WHERE id = $1`, grp).Scan(&status, &userNull, &queueKept))
	assert.Equal(t, "open", status, "grupo não tem aceite: continua aberto")
	assert.True(t, userNull)
	require.NoError(t, e.admin.QueryRow(`SELECT status, "userId" IS NULL, true FROM "Tickets" WHERE id = $1`, closed).Scan(&status, &userNull, &queueKept))
	assert.Equal(t, "closed", status)
	assert.True(t, userNull)
}

// A carteira exige destino; com destino, passa inteira; o próprio usuário e outra empresa não são destino.
func TestDeleteUser_Carteira(t *testing.T) {
	e := newQEnv(t)
	u, dest, c := e.user(t, "u1@x"), e.user(t, "u2@x"), e.conn(t, "A")
	ct := e.one(t, `INSERT INTO "Contacts" (name, number, "tenantId", "connectionId", "walletUserId") VALUES ('c','1',$1,$2,$3) RETURNING id`, e.tenant, c, u)

	var w *services.UserWalletError
	require.ErrorAs(t, e.deleteUser(u, nil), &w)
	assert.EqualValues(t, 1, w.Contacts)
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Users" WHERE id = $1`, u), "nada mudou")

	assert.ErrorIs(t, e.deleteUser(u, &u), services.ErrWalletTargetInvalid, "ele mesmo")
	gone := 999999
	assert.ErrorIs(t, e.deleteUser(u, &gone), services.ErrWalletTargetInvalid, "usuário que não existe")

	require.NoError(t, e.deleteUser(u, &dest))
	assert.Equal(t, dest, e.count(t, `SELECT "walletUserId" FROM "Contacts" WHERE id = $1`, ct))
}

func TestDeleteUser_OutraEmpresaNaoSeExclui(t *testing.T) {
	e := newQEnv(t)
	other := uuid.New()
	_, err := e.admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'B','active',now(),now())`, other)
	require.NoError(t, err)
	foreign := e.one(t, `INSERT INTO "Users" (name, email, "passwordHash", "tenantId") VALUES ('x','x@x','h',$1) RETURNING id`, other)

	err = e.deleteUser(foreign, nil)
	assert.True(t, errors.Is(err, services.ErrUserNotFound))
	assert.Equal(t, 1, e.count(t, `SELECT count(*) FROM "Users" WHERE id = $1`, foreign))
}

// Vínculos de fila e de setor, e o histórico que aponta o usuário, saem junto; o responsável de uma chamada vira vazio (a chamada fica).
func TestDeleteUser_RemoveVinculosEHistorico(t *testing.T) {
	e := newQEnv(t)
	q, u, c := e.queue(t, "fila"), e.user(t, "u1@x"), e.conn(t, "A")
	tk := e.ticket(t, "wa:1", c, &q, nil, "open", false)
	_, err := e.admin.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES ($1,$2)`, u, q)
	require.NoError(t, err)
	setor := e.one(t, `INSERT INTO "Setores" (name, "tenantId") VALUES ('s',$1) RETURNING id`, e.tenant)
	_, err = e.admin.Exec(`INSERT INTO user_setores ("userId", "setorId", "ehGestor") VALUES ($1,$2,true)`, u, setor)
	require.NoError(t, err)
	_, err = e.admin.Exec(`INSERT INTO "TicketLogs" ("ticketId", "userId", type, payload, "tenantId") VALUES ($1,$2,'status','{}',$3)`, tk, u, e.tenant)
	require.NoError(t, err)
	_, err = e.admin.Exec(`INSERT INTO "CallRecordingAccesses" ("tenantId", "callId", "userId", action, at) VALUES ($1,'C1',$2,'listen',now())`, e.tenant, u)
	require.NoError(t, err)

	require.NoError(t, e.deleteUser(u, nil))

	for _, tbl := range []string{`user_queues WHERE user_id`, `user_setores WHERE "userId"`, `"TicketLogs" WHERE "userId"`, `"CallRecordingAccesses" WHERE "userId"`} {
		assert.Zero(t, e.count(t, `SELECT count(*) FROM `+tbl+` = $1`, u), tbl)
	}
}
