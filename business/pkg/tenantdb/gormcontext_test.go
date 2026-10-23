package tenantdb_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// O código de produção usa db.Session(&gorm.Session{NewDB: true}) em ~300 pontos. O isolamento só vale se esse padrão
// preservar o escopo de empresa do handle de origem. Este teste prova que preserva (e que um handle sem ctx não herda).
func TestRLS_SessionNewDBPreservaOEscopoDoHandle(t *testing.T) {
	_, db, a, _ := setup(t)

	scoped, done := tenantdb.TenantCtx(context.Background(), db, a)
	defer done()
	h := db.WithContext(scoped) // como o IsAuth entrega ao controller

	var got []ticket
	require.NoError(t, h.Session(&gorm.Session{NewDB: true}).Find(&got).Error)
	require.Len(t, got, 1)
	assert.Equal(t, a, got[0].TenantID)

	// E o handle global, sem escopo, continua recusado.
	require.Error(t, db.Session(&gorm.Session{NewDB: true}).Find(&got).Error)
}

func TestRLS_TransactionEPreloadHerdamOEscopo(t *testing.T) {
	_, db, a, _ := setup(t)

	scoped, done := tenantdb.TenantCtx(context.Background(), db, a)
	defer done()
	h := db.WithContext(scoped)

	require.NoError(t, h.Session(&gorm.Session{NewDB: true}).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&ticket{}).Count(&n).Error; err != nil {
			return err
		}
		assert.EqualValues(t, 1, n)
		return nil
	}))
}

func TestRLS_ReleaseDevolveAConexaoEORetomaSozinho(t *testing.T) {
	e, db, a, _ := setup(t)
	e.app.SetMaxOpenConns(1)

	scoped, done := tenantdb.TenantCtx(context.Background(), db, a)
	defer done()
	h := db.WithContext(scoped)

	var n int64
	require.NoError(t, h.Model(&ticket{}).Count(&n).Error)
	require.EqualValues(t, 1, n)

	tenantdb.Release(scoped)

	// Com 1 conexão no pool, outra requisição só consegue usar o banco se o Release devolveu a conexão.
	var other int
	require.NoError(t, e.app.QueryRow(`SELECT 1`).Scan(&other))

	// E o mesmo escopo volta a funcionar sozinho depois do Release.
	require.NoError(t, h.Model(&ticket{}).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}

func TestRLS_DetachNaoCompartilhaAConexaoComAGoroutine(t *testing.T) {
	_, db, a, _ := setup(t)

	scoped, done := tenantdb.TenantCtx(context.Background(), db, a)
	defer done()
	require.NoError(t, db.WithContext(scoped).Model(&ticket{}).Count(new(int64)).Error)

	bg, bgDone := tenantdb.Detach(scoped)
	res := make(chan int64, 1)
	go func() {
		defer bgDone()
		var n int64
		_ = db.WithContext(bg).Model(&ticket{}).Count(&n).Error
		res <- n
	}()
	assert.EqualValues(t, 1, <-res, "a goroutine enxerga só a empresa original, em conexão própria")
}

// Um handler que espera uma chamada externa lenta não pode segurar a conexão do banco. Com o pool limitado a UMA
// conexão, outra requisição só consegue usar o banco enquanto o primeiro espera se o Release devolveu a conexão.
func TestRLS_ReleaseLiberaOPoolDuranteChamadaExterna(t *testing.T) {
	e, db, a, _ := setup(t)
	e.app.SetMaxOpenConns(1)

	scoped, done := tenantdb.TenantCtx(context.Background(), db, a)
	defer done()
	h := db.WithContext(scoped)
	var n int64
	require.NoError(t, h.Model(&ticket{}).Count(&n).Error) // fixa a única conexão

	// Sem Release, o pool de 1 conexão está ocupado: outra requisição esgota o prazo.
	busy, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var x int
	assert.Error(t, e.app.QueryRowContext(busy, `SELECT 1`).Scan(&x), "conexão fixada: o pool está ocupado")

	tenantdb.Release(scoped) // início da espera pela chamada externa

	free, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	require.NoError(t, e.app.QueryRowContext(free, `SELECT 1`).Scan(&x), "depois do Release o pool está livre")

	require.NoError(t, h.Model(&ticket{}).Count(&n).Error) // fim da espera: o mesmo escopo retoma sozinho
	assert.EqualValues(t, 1, n)
}

// Sob carga concorrente, escopos de empresas diferentes nunca se enxergam: cada goroutine só vê a própria empresa,
// mesmo com o pool pequeno forçando a reutilização das mesmas conexões físicas entre empresas.
func TestRLS_EscoposConcorrentesNuncaSeVazam(t *testing.T) {
	e, db, a, b := setup(t)
	e.app.SetMaxOpenConns(3) // poucas conexões físicas, muita reutilização entre empresas

	const rounds = 60
	type result struct {
		want uuid.UUID
		got  []ticket
		err  error
	}
	out := make(chan result, rounds*2)
	var wg sync.WaitGroup
	for i := 0; i < rounds; i++ {
		for _, tn := range []uuid.UUID{a, b} {
			wg.Add(1)
			go func(tn uuid.UUID) {
				defer wg.Done()
				var got []ticket
				err := tenantdb.WithTenant(context.Background(), db, tn, func(ctx context.Context) error {
					return db.WithContext(ctx).Find(&got).Error // sem filtro algum
				})
				out <- result{want: tn, got: got, err: err}
			}(tn)
		}
	}
	wg.Wait()
	close(out)

	n := 0
	for r := range out {
		n++
		require.NoError(t, r.err)
		require.Len(t, r.got, 1, "cada empresa tem exatamente 1 ticket")
		assert.Equal(t, r.want, r.got[0].TenantID, "VAZAMENTO sob concorrência: empresa errada na resposta")
	}
	assert.Equal(t, rounds*2, n)
}
