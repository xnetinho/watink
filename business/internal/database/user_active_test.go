package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
)

// Desativar usuário (PR 1b) tira ele da distribuição: os testes abaixo rodam
// contra o banco migrado de verdade, com RLS no papel do app (sem BYPASSRLS).

func (e *qEnv) setActive(t *testing.T, userID int, active bool) {
	t.Helper()
	_, err := e.admin.Exec(`UPDATE "Users" SET active = $2 WHERE id = $1`, userID, active)
	require.NoError(t, err)
}

func (e *qEnv) addToQueue(t *testing.T, userID, queueID int) {
	t.Helper()
	_, err := e.admin.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES ($1,$2)`, userID, queueID)
	require.NoError(t, err)
}

// FindQueueUsers é a fonte de candidatos do rodízio/balanceamento: inativo não entra.
func TestFindQueueUsers_ExcluiUsuarioInativo(t *testing.T) {
	e := newQEnv(t)
	q := e.queue(t, "fila")
	ativo := e.user(t, "ativo@test.com")
	inativo := e.user(t, "inativo@test.com")
	e.setActive(t, inativo, false)
	e.addToQueue(t, ativo, q)
	e.addToQueue(t, inativo, q)

	repo := repository.NewGormUserQueueRepository(e.gdb)
	var users []domain.User
	err := tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		var err error
		users, err = repo.FindQueueUsers(ctx, q, e.tenant)
		return err
	})
	require.NoError(t, err)
	require.Len(t, users, 1, "só o usuário ativo pode ser candidato")
	assert.Equal(t, ativo, users[0].ID)
}

// A carteira também não pode devolver o ticket para um usuário desativado.
func TestIsUserInQueue_IgnoraUsuarioInativo(t *testing.T) {
	e := newQEnv(t)
	q := e.queue(t, "fila")
	ativo := e.user(t, "ativo@test.com")
	inativo := e.user(t, "inativo@test.com")
	e.setActive(t, inativo, false)
	e.addToQueue(t, ativo, q)
	e.addToQueue(t, inativo, q)

	repo := repository.NewGormUserQueueRepository(e.gdb)
	require.NoError(t, tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		in, err := repo.IsUserInQueue(ctx, ativo, q)
		require.NoError(t, err)
		assert.True(t, in, "membro ativo continua elegível")

		in, err = repo.IsUserInQueue(ctx, inativo, q)
		require.NoError(t, err)
		assert.False(t, in, "membro inativo não é elegível")
		return nil
	}))
}

// Reativar devolve o usuário à distribuição sem reconfigurar vínculos.
func TestFindQueueUsers_ReativadoVoltaAElegivel(t *testing.T) {
	e := newQEnv(t)
	q := e.queue(t, "fila")
	u := e.user(t, "volta@test.com")
	e.setActive(t, u, false)
	e.addToQueue(t, u, q)

	repo := repository.NewGormUserQueueRepository(e.gdb)
	err := tenantdb.WithTenant(context.Background(), e.gdb, e.tenant, func(ctx context.Context) error {
		users, err := repo.FindQueueUsers(ctx, q, e.tenant)
		require.NoError(t, err)
		assert.Empty(t, users)

		e.setActive(t, u, true)

		users, err = repo.FindQueueUsers(ctx, q, e.tenant)
		require.NoError(t, err)
		assert.Len(t, users, 1)
		return nil
	})
	require.NoError(t, err)
}
