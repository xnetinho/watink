package tenantdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	pwOwner  = "owner_pw_test"
	pwApp    = "app_pw_test"
	pwSystem = "system_pw_test"
)

// env guarda o banco descartável com os papéis reais usados por RLS.
type env struct {
	name   string
	super  *sql.DB // postgres (apenas para criar papéis/banco)
	owner  *sql.DB // watink_owner: dono dos objetos, roda DDL e políticas
	app    *sql.DB // watink_app: NOSUPERUSER NOBYPASSRLS
	system *sql.DB // watink_system: BYPASSRLS
	router *tenantdb.Router
}

func baseURL(t *testing.T) *url.URL {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func open(t *testing.T, u *url.URL, user, pass, db string) *sql.DB {
	t.Helper()
	c := *u
	c.User = url.UserPassword(user, pass)
	c.Path = "/" + db
	conn, err := sql.Open("pgx", c.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// newEnv cria os papéis (idempotente), um banco descartável e o esquema mínimo, e devolve os pools por papel.
func newEnv(t *testing.T) *env {
	t.Helper()
	u := baseURL(t)
	pw, _ := u.User.Password()
	super := open(t, u, u.User.Username(), pw, strings.TrimPrefix(u.Path, "/"))
	ctx := context.Background()

	var isSuper bool
	require.NoError(t, super.QueryRowContext(ctx, `SELECT rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&isSuper))
	if !isSuper {
		t.Skip("o usuário de TEST_DATABASE_URL precisa ser superusuário para criar papéis")
	}

	// Pacotes de teste rodam em paralelo e papéis são objetos do cluster: serializa a criação/alteração com um
	// advisory lock de sessão (sem ele, dois pacotes colidem com "tuple concurrently updated").
	adm := open(t, u, u.User.Username(), pw, "postgres")
	unlockCluster, err := tenantdb.LockCluster(ctx, adm)
	require.NoError(t, err)
	defer unlockCluster()

	for _, r := range []struct{ name, pass, attrs string }{
		{"watink_owner", pwOwner, "LOGIN NOSUPERUSER CREATEDB"},
		{"watink_app", pwApp, "LOGIN NOSUPERUSER NOBYPASSRLS"},
		{"watink_system", pwSystem, "LOGIN NOSUPERUSER BYPASSRLS"},
	} {
		// CREATE ROLE sem IF NOT EXISTS: ignora "já existe" e reaplica atributos e senha.
		_, _ = super.ExecContext(ctx, fmt.Sprintf(`CREATE ROLE %s`, r.name))
		_, err := super.ExecContext(ctx, fmt.Sprintf(`ALTER ROLE %s %s PASSWORD '%s'`, r.name, r.attrs, r.pass))
		require.NoError(t, err)
	}

	name := "rls_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	_, err = super.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER watink_owner`, name))
	require.NoError(t, err)
	t.Cleanup(func() {
		// DROP DATABASE também mexe nos catálogos compartilhados (dependências de papéis): mesmo lock.
		if unlock, err := tenantdb.LockCluster(context.Background(), adm); err == nil {
			defer unlock()
		}
		_, _ = super.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name))
	})

	e := &env{name: name, super: super}
	e.owner = open(t, u, "watink_owner", pwOwner, name)
	e.app = open(t, u, "watink_app", pwApp, name)
	e.system = open(t, u, "watink_system", pwSystem, name)

	// Privilégio mínimo: o app e o sistema só fazem DML.
	for _, q := range []string{
		`REVOKE ALL ON SCHEMA public FROM PUBLIC`,
		`GRANT USAGE ON SCHEMA public TO watink_app, watink_system`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE watink_owner IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO watink_app, watink_system`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE watink_owner IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO watink_app, watink_system`,
	} {
		_, err := e.owner.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}
	return e
}

// schema cria tabelas de exemplo: com tenantId, filha e global.
func (e *env) schema(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE "Tenants" (id uuid PRIMARY KEY, name text NOT NULL)`,
		`CREATE TABLE "Plans" (id serial PRIMARY KEY, name text)`,
		`CREATE TABLE "Pipelines" (id serial PRIMARY KEY, "tenantId" uuid NOT NULL, name text)`,
		`CREATE TABLE "PipelineStages" (id serial PRIMARY KEY, "pipelineId" int NOT NULL REFERENCES "Pipelines"(id) ON DELETE CASCADE, name text)`,
		`CREATE TABLE "Tickets" (id serial PRIMARY KEY, "tenantId" uuid NOT NULL, status text, code text UNIQUE)`,
	} {
		_, err := e.owner.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}
}

// guard aplica o conjunto de políticas pelo código de produção, como dono.
func (e *env) guard(t *testing.T) {
	t.Helper()
	require.NoError(t, tenantdb.ApplyPolicies(context.Background(), e.owner))
	var err error
	e.router, err = tenantdb.NewRouter(e.app, e.system)
	require.NoError(t, err)
}
