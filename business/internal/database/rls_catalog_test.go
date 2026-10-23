package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Os papéis são objetos do CLUSTER, compartilhados por todos os testes (e pelo pacote pkg/tenantdb). Todos precisam usar
// as MESMAS senhas: com valores diferentes, testes em paralelo sobrescrevem a senha uns dos outros.
const (
	testAppPass    = "app_pw_test"
	testSystemPass = "system_pw_test"
	testEnginePass = "engine_pw_test"
)

// migratedDB roda o job de migração REAL (RunMigrations) num banco descartável e devolve os pools por papel. É o teste
// que prova a configuração de produção: papéis, esquema completo, políticas e privilégios.
func migratedDB(t *testing.T) (admin, app, system *sql.DB, name string) {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	u, err := url.Parse(raw)
	require.NoError(t, err)
	pw, _ := u.User.Password()

	root, err := sql.Open("pgx", raw)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })
	var super bool
	require.NoError(t, root.QueryRow(`SELECT rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&super))
	if !super {
		t.Skip("precisa de superusuário para criar o banco e os papéis")
	}

	name = "mig_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	// CREATE/DROP DATABASE tocam catálogos compartilhados do cluster: o mesmo lock do job de migração e dos papéis.
	adm := openAs(t, u, u.User.Username(), pw, "postgres")
	unlockCreate, err := tenantdb.LockCluster(context.Background(), adm)
	require.NoError(t, err)
	_, err = root.Exec(`CREATE DATABASE ` + name)
	unlockCreate()
	require.NoError(t, err)
	t.Cleanup(func() {
		if unlock, err := tenantdb.LockCluster(context.Background(), adm); err == nil {
			defer unlock()
		}
		_, _ = root.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`)
	})
	for _, ext := range []string{"vector", "postgis"} {
		ec, err := sql.Open("pgx", withDB(u, name))
		require.NoError(t, err)
		_, _ = ec.Exec(`CREATE EXTENSION IF NOT EXISTS ` + ext)
		_ = ec.Close()
	}

	host, port := u.Hostname(), u.Port()
	t.Setenv("DB_HOST", host)
	t.Setenv("DB_PORT", port)
	t.Setenv("DB_NAME", name)
	t.Setenv("DB_ADMIN_USER", u.User.Username())
	t.Setenv("DB_ADMIN_PASS", pw)
	t.Setenv("DB_APP_PASS", testAppPass)
	t.Setenv("DB_SYSTEM_PASS", testSystemPass)
	prev := DB
	t.Cleanup(func() { DB = prev })

	require.NoError(t, RunMigrations())

	admin = openAs(t, u, u.User.Username(), pw, name)
	app = openAs(t, u, RoleApp, testAppPass, name)
	system = openAs(t, u, RoleSystem, testSystemPass, name)
	return admin, app, system, name
}

func withDB(u *url.URL, db string) string {
	c := *u
	c.Path = "/" + db
	return c.String()
}

func openAs(t *testing.T, u *url.URL, user, pass, db string) *sql.DB {
	t.Helper()
	c := *u
	c.User = url.UserPassword(user, pass)
	c.Path = "/" + db
	conn, err := sql.Open("pgx", c.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestMigracao_PapeisEPrivilegios(t *testing.T) {
	_, app, system, _ := migratedDB(t)

	var super, bypass bool
	require.NoError(t, app.QueryRow(`SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&super, &bypass))
	assert.False(t, super)
	assert.False(t, bypass, "o papel do app não pode ignorar RLS")

	require.NoError(t, system.QueryRow(`SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&super, &bypass))
	assert.False(t, super, "o papel de sistema não é superusuário")
	assert.True(t, bypass)

	for _, q := range []string{
		`ALTER TABLE "Tickets" DISABLE ROW LEVEL SECURITY`,
		`DROP POLICY tenant_isolation ON "Tickets"`,
		`TRUNCATE "Tickets"`,
		`DROP TABLE "Users"`,
		`CREATE TABLE intruso (id int)`,
	} {
		_, err := app.Exec(q)
		assert.Error(t, err, "o app não pode: %s", q)
		_, err = system.Exec(q)
		assert.Error(t, err, "o sistema não pode: %s", q)
	}
}

func TestMigracao_Idempotente(t *testing.T) {
	admin, _, _, _ := migratedDB(t)
	require.NoError(t, RunMigrations(), "segunda execução não pode falhar")
	problems, err := tenantdb.Verify(context.Background(), admin)
	require.NoError(t, err)
	assert.Empty(t, problems)
}

func TestCatalogo_TodaTabelaComTenantIdTemPoliticaAtiva(t *testing.T) {
	admin, _, _, _ := migratedDB(t)
	problems, err := tenantdb.Verify(context.Background(), admin)
	require.NoError(t, err)
	assert.Empty(t, problems)

	cat, err := tenantdb.ReadCatalog(context.Background(), admin)
	require.NoError(t, err)
	assert.Empty(t, cat.Unclassified(), "toda tabela sem tenantId precisa estar classificada")
	assert.Greater(t, len(cat.WithTenant), 50, "o catálogo real tem dezenas de tabelas com tenantId")
}

// A empresa A nunca lê, altera nem apaga linha da B em NENHUMA tabela com tenantId: sem filtro algum na consulta.
func TestVazamento_PorTabelaGeradoDoCatalogo(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	ctx := context.Background()
	cat, err := tenantdb.ReadCatalog(ctx, admin)
	require.NoError(t, err)

	a, b := uuid.New(), uuid.New()
	_, err = admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now()), ($2,'B','active',now(),now())`, a, b)
	require.NoError(t, err)

	// Linhas mínimas por tabela: o admin ignora RLS e semeia as duas empresas. Tabelas com colunas NOT NULL sem default
	// que não dá para preencher genericamente são puladas e listadas, para a cobertura ficar visível.
	var seeded, skipped []string
	for _, tbl := range cat.WithTenant {
		if tbl == "Tenants" {
			continue
		}
		ok := true
		for _, tn := range []uuid.UUID{a, b} {
			if err := seedRow(admin, tbl, tn); err != nil {
				ok = false
				break
			}
		}
		if ok {
			seeded = append(seeded, tbl)
		} else {
			skipped = append(skipped, tbl)
		}
	}
	require.NotEmpty(t, seeded)
	t.Logf("tabelas verificadas: %d; puladas (colunas obrigatórias sem default): %d %v", len(seeded), len(skipped), skipped)
	assert.Greater(t, len(seeded), len(skipped), "a maioria das tabelas precisa ser exercitada")

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	for _, tbl := range seeded {
		var seenA, seenB int64
		require.NoError(t, tenantdb.WithTenant(ctx, gdb, a, func(c context.Context) error {
			if err := gdb.WithContext(c).Table(tbl).Where(`"tenantId" = ?`, a).Count(&seenA).Error; err != nil {
				return err
			}
			return gdb.WithContext(c).Table(tbl).Where(`"tenantId" = ?`, b).Count(&seenB).Error
		}), tbl)
		assert.EqualValues(t, 1, seenA, "%s: a empresa A deve ver a própria linha", tbl)
		assert.Zero(t, seenB, "VAZAMENTO em %s: A viu linha da B", tbl)

		// Sem filtro algum, A só enxerga a si mesma.
		var all int64
		require.NoError(t, tenantdb.WithTenant(ctx, gdb, a, func(c context.Context) error {
			return gdb.WithContext(c).Table(tbl).Count(&all).Error
		}), tbl)
		assert.EqualValues(t, 1, all, "VAZAMENTO em %s: consulta sem filtro devolveu mais que a linha da A", tbl)

		// Escrita para a B, estando no escopo da A, é recusada.
		err := tenantdb.WithTenant(ctx, gdb, a, func(c context.Context) error {
			return gdb.WithContext(c).Exec(fmt.Sprintf(`UPDATE %q SET "tenantId" = ? WHERE "tenantId" = ?`, tbl), b, a).Error
		})
		assert.Error(t, err, "%s: A não pode reatribuir linha para a B", tbl)

		// Sem empresa definida o banco não expõe nada, mesmo contornando o roteador.
		var raw int
		require.NoError(t, app.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %q`, tbl)).Scan(&raw), tbl)
		assert.Zero(t, raw, "%s: sem empresa definida o banco deve devolver zero linhas", tbl)
	}
}

func routerOver(t *testing.T, app *sql.DB) *tenantdb.Router {
	t.Helper()
	r, err := tenantdb.NewRouter(app, nil)
	require.NoError(t, err)
	return r
}

// seedRow insere uma linha mínima com a empresa informada, preenchendo colunas NOT NULL sem default por tipo.
func seedRow(db *sql.DB, table string, tenant uuid.UUID) error {
	rows, err := db.Query(`
		SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND is_nullable = 'NO' AND column_default IS NULL
		  AND is_identity = 'NO' AND is_generated = 'NEVER' ORDER BY ordinal_position`, table)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var cols, vals []string
	var args []any
	hasTenant := false
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return err
		}
		cols = append(cols, fmt.Sprintf("%q", name))
		vals = append(vals, fmt.Sprintf("$%d", len(cols)))
		switch {
		case name == "tenantId":
			args = append(args, tenant)
			hasTenant = true
		case typ == "uuid":
			args = append(args, uuid.New())
		case typ == "boolean":
			args = append(args, false)
		case strings.Contains(typ, "int"), typ == "numeric", typ == "double precision", typ == "real":
			args = append(args, 1)
		case strings.HasPrefix(typ, "timestamp"), typ == "date":
			args = append(args, "2026-01-01")
		case typ == "jsonb" || typ == "json":
			args = append(args, "{}")
		default:
			args = append(args, "x_"+uuid.NewString()[:8])
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasTenant {
		cols = append(cols, `"tenantId"`)
		vals = append(vals, fmt.Sprintf("$%d", len(cols)))
		args = append(args, tenant)
	}
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO %q (%s) VALUES (%s)`, table, strings.Join(cols, ","), strings.Join(vals, ",")), args...)
	return err
}

// O papel do engine só lê Whatsapps: nenhuma outra tabela do negócio, nenhum DML e nenhum DDL em public.
func TestMigracao_PapelDoEngineTemPrivilegioMinimo(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	t.Setenv("DB_ENGINE_PASS", testEnginePass)
	_, _, _, name := migratedDB(t)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	engine := openAs(t, u, RoleEngine, testEnginePass, name)

	var n int
	require.NoError(t, engine.QueryRow(`SELECT count(*) FROM public."Whatsapps"`).Scan(&n), "SELECT em Whatsapps é permitido")

	for _, q := range []string{
		`SELECT count(*) FROM public."Tickets"`,
		`SELECT count(*) FROM public."Users"`,
		`SELECT count(*) FROM public."Messages"`,
		`SELECT count(*) FROM public."Contacts"`,
		`UPDATE public."Whatsapps" SET name = name`,
		`DELETE FROM public."Whatsapps"`,
		`CREATE TABLE public.intruso (i int)`,
	} {
		_, err := engine.Exec(q)
		assert.Error(t, err, "o engine não pode: %s", q)
	}

	var owner string
	require.NoError(t, engine.QueryRow(`SELECT nspowner::regrole::text FROM pg_namespace WHERE nspname = 'whatsmeow'`).Scan(&owner))
	assert.Equal(t, RoleEngine, owner, "o schema whatsmeow pertence ao engine")
	_, err = engine.Exec(`CREATE TABLE whatsmeow.probe (i int)`)
	assert.NoError(t, err, "o engine cria tabelas no próprio schema")
}
