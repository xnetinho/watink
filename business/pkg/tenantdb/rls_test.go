package tenantdb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ticket struct {
	ID       int       `gorm:"primaryKey"`
	TenantID uuid.UUID `gorm:"column:tenantId"`
	Status   string
	Code     *string
}

func (ticket) TableName() string { return "Tickets" }

type pipeline struct {
	ID       int       `gorm:"primaryKey"`
	TenantID uuid.UUID `gorm:"column:tenantId"`
	Name     string
}

func (pipeline) TableName() string { return "Pipelines" }

type stage struct {
	ID         int `gorm:"primaryKey"`
	PipelineID int `gorm:"column:pipelineId"`
	Name       string
}

func (stage) TableName() string { return "PipelineStages" }

func gormOver(t *testing.T, r *tenantdb.Router) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: r}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	return db
}

func setup(t *testing.T) (*env, *gorm.DB, uuid.UUID, uuid.UUID) {
	e := newEnv(t)
	e.schema(t)
	e.guard(t)
	db := gormOver(t, e.router)
	a, b := uuid.New(), uuid.New()
	// Sementes pelo caminho de sistema (BYPASSRLS), como faz o provisionamento.
	require.NoError(t, tenantdb.WithSystem(context.Background(), db, func(ctx context.Context) error {
		for _, tn := range []uuid.UUID{a, b} {
			if err := db.WithContext(ctx).Create(&ticket{TenantID: tn, Status: "open"}).Error; err != nil {
				return err
			}
		}
		return nil
	}))
	return e, db, a, b
}

func TestRLS_ConsultaSemFiltroVeSoAEmpresaDoEscopo(t *testing.T) {
	_, db, a, b := setup(t)

	var got []ticket
	err := tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		return db.WithContext(ctx).Find(&got).Error // sem WHERE de empresa
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, a, got[0].TenantID)
	assert.NotEqual(t, b, got[0].TenantID, "VAZAMENTO: linha da empresa B visível no escopo da A")
}

func TestRLS_EscritaParaOutraEmpresaERecusada(t *testing.T) {
	_, db, a, b := setup(t)

	err := tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		return db.WithContext(ctx).Create(&ticket{TenantID: b, Status: "forjado"}).Error
	})
	require.Error(t, err, "inserir com a empresa B no escopo da A deve falhar")
	assert.Contains(t, err.Error(), "row-level security")

	// E nada foi gravado.
	var n int64
	require.NoError(t, tenantdb.WithSystem(context.Background(), db, func(ctx context.Context) error {
		return db.WithContext(ctx).Model(&ticket{}).Where("status = ?", "forjado").Count(&n).Error
	}))
	assert.Zero(t, n)
}

func TestRLS_UpdateEDeleteNaoAtingemOutraEmpresa(t *testing.T) {
	_, db, a, b := setup(t)

	require.NoError(t, tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		res := db.WithContext(ctx).Model(&ticket{}).Where("1 = 1").Update("status", "alterado")
		assert.EqualValues(t, 1, res.RowsAffected, "só a linha da própria empresa")
		res = db.WithContext(ctx).Where("1 = 1").Delete(&ticket{})
		assert.EqualValues(t, 1, res.RowsAffected)
		return res.Error
	}))

	var restante ticket
	require.NoError(t, tenantdb.WithSystem(context.Background(), db, func(ctx context.Context) error {
		return db.WithContext(ctx).Where(`"tenantId" = ?`, b).First(&restante).Error
	}))
	assert.Equal(t, "open", restante.Status, "linha da empresa B intacta")
}

func TestRLS_SemEscopoFalhaRuidosaEmTabelaDeEmpresa(t *testing.T) {
	_, db, _, _ := setup(t)

	var got []ticket
	err := db.WithContext(context.Background()).Find(&got).Error
	require.Error(t, err)
	assert.True(t, errors.Is(err, tenantdb.ErrNoScope), "esperava ErrNoScope, veio %v", err)
}

func TestRLS_SemEscopoTabelaGlobalFunciona(t *testing.T) {
	_, db, _, _ := setup(t)
	var n int64
	require.NoError(t, db.WithContext(context.Background()).Table(`"Plans"`).Count(&n).Error)
}

func TestRLS_BancoFalhaFechadoMesmoSemOGuardaDoRouter(t *testing.T) {
	e, _, _, _ := setup(t)

	// Conexão crua do papel do app, sem app.tenant_id: o banco sozinho devolve zero linhas.
	var n int
	require.NoError(t, e.app.QueryRow(`SELECT count(*) FROM "Tickets"`).Scan(&n))
	assert.Zero(t, n, "sem empresa definida o banco não pode expor nenhuma linha")

	_, err := e.app.Exec(`INSERT INTO "Tickets" ("tenantId", status) VALUES (gen_random_uuid(), 'x')`)
	require.Error(t, err, "sem empresa definida nenhuma escrita é permitida")
}

func TestRLS_EmpresaNaoVazaParaAProximaRequisicaoDoPool(t *testing.T) {
	e, db, a, _ := setup(t)
	e.app.SetMaxOpenConns(1) // força a mesma conexão física entre os dois usos

	require.NoError(t, tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		var n int64
		return db.WithContext(ctx).Model(&ticket{}).Count(&n).Error
	}))

	var setting string
	require.NoError(t, e.app.QueryRow(`SELECT coalesce(current_setting('app.tenant_id', true), '')`).Scan(&setting))
	assert.Empty(t, setting, "a empresa precisa ser limpa ao devolver a conexão")

	var n int
	require.NoError(t, e.app.QueryRow(`SELECT count(*) FROM "Tickets"`).Scan(&n))
	assert.Zero(t, n, "a conexão reutilizada não pode herdar a empresa anterior")
}

func TestRLS_TabelaFilhaHerdaIsolamentoDoPai(t *testing.T) {
	_, db, a, b := setup(t)

	var pa, pb pipeline
	require.NoError(t, tenantdb.WithSystem(context.Background(), db, func(ctx context.Context) error {
		pa, pb = pipeline{TenantID: a, Name: "A"}, pipeline{TenantID: b, Name: "B"}
		if err := db.WithContext(ctx).Create(&pa).Error; err != nil {
			return err
		}
		if err := db.WithContext(ctx).Create(&pb).Error; err != nil {
			return err
		}
		if err := db.WithContext(ctx).Create(&stage{PipelineID: pa.ID, Name: "sa"}).Error; err != nil {
			return err
		}
		return db.WithContext(ctx).Create(&stage{PipelineID: pb.ID, Name: "sb"}).Error
	}))

	var got []stage
	require.NoError(t, tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		return db.WithContext(ctx).Find(&got).Error
	}))
	require.Len(t, got, 1)
	assert.Equal(t, "sa", got[0].Name)

	err := tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		return db.WithContext(ctx).Create(&stage{PipelineID: pb.ID, Name: "invasor"}).Error
	})
	require.Error(t, err, "filho de pipeline de outra empresa deve ser recusado")
}

func TestRLS_PapelDoAppNaoAlteraOEsquema(t *testing.T) {
	e, _, _, _ := setup(t)

	for _, q := range []string{
		`ALTER TABLE "Tickets" DISABLE ROW LEVEL SECURITY`,
		`DROP POLICY tenant_isolation ON "Tickets"`,
		`TRUNCATE "Tickets"`,
		`DROP TABLE "Tickets"`,
		`SET ROLE watink_owner`,
		`CREATE TABLE intruso (id int)`,
	} {
		_, err := e.app.Exec(q)
		assert.Error(t, err, "o papel do app não pode executar: %s", q)
	}
}

func TestRLS_ErroDentroDaTransacaoDaRequisicaoNaoEnvenenaSeUsarSavepoint(t *testing.T) {
	_, db, a, _ := setup(t)

	err := tenantdb.WithTenant(context.Background(), db, a, func(ctx context.Context) error {
		tx := db.WithContext(ctx)
		return tx.Transaction(func(tx *gorm.DB) error {
			code := "dup"
			if err := tx.Create(&ticket{TenantID: a, Status: "open", Code: &code}).Error; err != nil {
				return err
			}
			// Duplicata tratada como "já existe" dentro de um savepoint, o fluxo segue.
			dupErr := tx.Transaction(func(sp *gorm.DB) error {
				return sp.Create(&ticket{TenantID: a, Status: "open", Code: &code}).Error
			})
			require.Error(t, dupErr)
			var n int64
			return tx.Model(&ticket{}).Count(&n).Error // não pode falhar com "transaction is aborted"
		})
	})
	require.NoError(t, err)
}

func TestRLS_EscopoEncerradoRecusaUso(t *testing.T) {
	e, db, a, _ := setup(t)
	s := e.router.NewScope(a)
	ctx := tenantdb.WithScope(context.Background(), s)
	s.Close()

	var got []ticket
	err := db.WithContext(ctx).Find(&got).Error
	require.Error(t, err)
	assert.True(t, errors.Is(err, tenantdb.ErrScopeClosed))
}

func TestRLS_EscopoDeSistemaVeTodasAsEmpresas(t *testing.T) {
	_, db, _, _ := setup(t)
	var n int64
	require.NoError(t, tenantdb.WithSystem(context.Background(), db, func(ctx context.Context) error {
		return db.WithContext(ctx).Model(&ticket{}).Count(&n).Error
	}))
	assert.EqualValues(t, 2, n)
}

func TestCatalogo_TabelaNovaSemClassificacaoFalha(t *testing.T) {
	e := newEnv(t)
	e.schema(t)
	_, err := e.owner.Exec(`CREATE TABLE "Orfa" (id serial PRIMARY KEY)`)
	require.NoError(t, err)

	err = tenantdb.ApplyPolicies(context.Background(), e.owner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Orfa")
}

func TestCatalogo_VerifyApontaTabelaSemPolitica(t *testing.T) {
	e := newEnv(t)
	e.schema(t)
	e.guard(t)

	problems, err := tenantdb.Verify(context.Background(), e.owner)
	require.NoError(t, err)
	assert.Empty(t, problems)

	_, err = e.owner.Exec(`ALTER TABLE "Tickets" DISABLE ROW LEVEL SECURITY`)
	require.NoError(t, err)
	problems, err = tenantdb.Verify(context.Background(), e.owner)
	require.NoError(t, err)
	assert.Contains(t, problems, "RLS desligado: Tickets")
}
