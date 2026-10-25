package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Papéis de banco (ADR 0034). O aplicativo nunca usa credencial de administrador.
const (
	RoleApp    = "watink_app"    // NOSUPERUSER NOBYPASSRLS: toda requisição e todo evento com empresa
	RoleSystem = "watink_system" // NOSUPERUSER BYPASSRLS: login, webhooks, varreduras (lista fechada)
)

var (
	// Router roteia cada comando pelo escopo (empresa ou sistema) do contexto. Nil antes de Connect.
	Router *tenantdb.Router
	// poolApp e poolSystem são os pools por trás do Router.
	poolApp, poolSystem *sql.DB
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}

func dsnFor(user, pass string) string { return dsnForDB(user, pass, os.Getenv("DB_NAME")) }

func dsnForDB(user, pass, dbname string) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"), user, pass, dbname, os.Getenv("DB_PORT"))
}

// openPool abre um pool pelo mesmo driver do GORM (mantém o tratamento de fuso e tipos) e aplica limites.
func openPool(user, pass string, maxOpen int) (*sql.DB, error) {
	g, err := gorm.Open(postgres.Open(dsnFor(user, pass)), &gorm.Config{Logger: newGormLogger()})
	if err != nil {
		return nil, err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxOpen / 2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return sqlDB, nil
}

// Connect abre os dois pools (app e sistema) e instala o Router como a conexão do handle global DB.
// Falha se o papel do aplicativo for superusuário ou ignorar RLS: nesse caso o isolamento seria só de fachada.
func Connect() {
	appPass := os.Getenv("DB_APP_PASS")
	sysPass := os.Getenv("DB_SYSTEM_PASS")
	if appPass == "" || sysPass == "" {
		log.Fatalf("DB_APP_PASS e DB_SYSTEM_PASS são obrigatórias (papéis %s e %s)", RoleApp, RoleSystem)
	}

	var err error
	poolApp, err = openPool(env("DB_APP_USER", RoleApp), appPass, envInt("DB_MAX_OPEN_APP", 60))
	if err != nil {
		log.Fatalf("Failed to connect to database (app): %v", err)
	}
	poolSystem, err = openPool(env("DB_SYSTEM_USER", RoleSystem), sysPass, envInt("DB_MAX_OPEN_SYSTEM", 10))
	if err != nil {
		log.Fatalf("Failed to connect to database (system): %v", err)
	}
	if err := assertAppRoleIsRestricted(context.Background(), poolApp); err != nil {
		log.Fatalf("Papel do aplicativo inseguro: %v", err)
	}

	Router, err = tenantdb.NewRouter(poolApp, poolSystem)
	if err != nil {
		log.Fatalf("Failed to initialize tenant router (rode `watink-business migrate` antes): %v", err)
	}
	DB, err = gorm.Open(postgres.New(postgres.Config{Conn: Router}), &gorm.Config{Logger: newGormLogger()})
	if err != nil {
		log.Fatalf("Failed to open gorm over router: %v", err)
	}
	fmt.Println("Connected to database successfully")
}

// assertAppRoleIsRestricted recusa um papel de aplicativo capaz de ignorar RLS ou de alterar o esquema.
func assertAppRoleIsRestricted(ctx context.Context, db *sql.DB) error {
	var user string
	var super, bypass bool
	err := db.QueryRowContext(ctx, `SELECT current_user, r.rolsuper, r.rolbypassrls FROM pg_roles r WHERE r.rolname = current_user`).
		Scan(&user, &super, &bypass)
	if err != nil {
		return err
	}
	if super || bypass {
		return fmt.Errorf("%q é superusuário ou BYPASSRLS; use o papel %s", user, RoleApp)
	}
	return nil
}

// Close encerra os pools.
func Close() {
	if poolApp != nil {
		_ = poolApp.Close()
	}
	if poolSystem != nil {
		_ = poolSystem.Close()
	}
}

// RunMigrations é o job de migração: roda como administrador (nunca dentro do servidor), cria os papéis, o esquema, as
// políticas RLS e os privilégios mínimos. É idempotente. Disparado por `watink-business migrate`.
func RunMigrations() error {
	adminUser := env("DB_ADMIN_USER", "postgres")
	adminPass := env("DB_ADMIN_PASS", os.Getenv("DB_PASS"))
	appPass := os.Getenv("DB_APP_PASS")
	sysPass := os.Getenv("DB_SYSTEM_PASS")
	if appPass == "" || sysPass == "" {
		return errors.New("DB_APP_PASS e DB_SYSTEM_PASS são obrigatórias")
	}
	enginePass := os.Getenv("DB_ENGINE_PASS") // opcional: sem ela o papel do engine não é criado

	g, err := gorm.Open(postgres.Open(dsnFor(adminUser, adminPass)), &gorm.Config{Logger: newGormLogger()})
	if err != nil {
		return fmt.Errorf("conectar como administrador: %w", err)
	}
	admin, err := g.DB()
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close() }()

	// Advisory lock: duas instâncias nunca migram ao mesmo tempo.
	ctx := context.Background()
	conn, err := admin.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	// 727274 serializa migrações do mesmo banco.
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(727274)`); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(ctx, `SELECT pg_advisory_unlock(727274)`) }()

	// Papéis, privilégios e políticas são objetos do CLUSTER: duas migrações simultâneas (inclusive de bancos diferentes,
	// como nos testes em paralelo) colidem com "tuple concurrently updated". Um único lock cobre a migração INTEIRA,
	// sem janela entre etapas.
	unlockCluster, err := tenantdb.LockClusterVia(ctx, func(dbname string) (*sql.DB, error) {
		g, err := gorm.Open(postgres.Open(dsnForDB(adminUser, adminPass, dbname)), &gorm.Config{Logger: newGormLogger()})
		if err != nil {
			return nil, err
		}
		return g.DB()
	})
	if err != nil {
		return err
	}
	defer unlockCluster()

	if err := ensureRoles(ctx, admin, appPass, sysPass); err != nil {
		return err
	}

	DB = g
	if err := MigrateVersioned(ctx, g); err != nil {
		return fmt.Errorf("baseline do esquema: %w", err)
	}

	if err := tenantdb.ApplyPoliciesLocked(ctx, admin); err != nil {
		return fmt.Errorf("políticas RLS: %w", err)
	}
	// O papel do engine precisa do GRANT em Whatsapps, então nasce depois do esquema.
	if enginePass != "" {
		if err := ensureEngineRole(ctx, admin, enginePass); err != nil {
			return err
		}
	}
	if err := grantMinimal(ctx, admin); err != nil {
		return err
	}
	problems, err := tenantdb.Verify(ctx, admin)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("isolamento incompleto: %v", problems)
	}
	fmt.Println("Migration job finished")
	return nil
}

func ensureRoles(ctx context.Context, admin *sql.DB, appPass, sysPass string) error {
	for _, r := range []struct{ name, pass, attrs string }{
		{env("DB_APP_USER", RoleApp), appPass, "LOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE"},
		{env("DB_SYSTEM_USER", RoleSystem), sysPass, "LOGIN NOSUPERUSER BYPASSRLS NOCREATEDB NOCREATEROLE"},
	} {
		var exists bool
		if err := admin.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, r.name).Scan(&exists); err != nil {
			return err
		}
		verb := "CREATE"
		if exists {
			verb = "ALTER"
		}
		// Nome e atributos vêm do código/ambiente do operador; a senha é literal escapada pelo driver abaixo.
		stmt := fmt.Sprintf(`%s ROLE %s %s PASSWORD %s`, verb, quoteIdent(r.name), r.attrs, quoteLiteral(r.pass))
		if _, err := admin.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("papel %s: %w", r.name, err)
		}
	}
	return nil
}

// RoleEngine é o papel do engine do WhatsApp (ADR 0034, D10): dono do schema "whatsmeow" (onde a biblioteca guarda chaves
// e sessões) e SELECT apenas em Whatsapps, para o reinício de sessões. Não lê nenhuma outra tabela do negócio.
const RoleEngine = "watink_engine"

func ensureEngineRole(ctx context.Context, admin *sql.DB, pass string) error {
	name := quoteIdent(env("DB_ENGINE_USER", RoleEngine))
	var exists bool
	if err := admin.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, env("DB_ENGINE_USER", RoleEngine)).Scan(&exists); err != nil {
		return err
	}
	verb := "CREATE"
	if exists {
		verb = "ALTER"
	}
	for _, q := range []string{
		fmt.Sprintf(`%s ROLE %s LOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE PASSWORD %s`, verb, name, quoteLiteral(pass)),
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS whatsmeow AUTHORIZATION %s`, name),
		fmt.Sprintf(`ALTER SCHEMA whatsmeow OWNER TO %s`, name),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, name),
		fmt.Sprintf(`GRANT SELECT ON public."Whatsapps" TO %s`, name),
		fmt.Sprintf(`ALTER ROLE %s SET search_path = whatsmeow`, name),
	} {
		if _, err := admin.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("papel do engine: %s: %w", q, err)
		}
	}
	return nil
}

func grantMinimal(ctx context.Context, admin *sql.DB) error {
	app, sys := quoteIdent(env("DB_APP_USER", RoleApp)), quoteIdent(env("DB_SYSTEM_USER", RoleSystem))
	var owner string
	if err := admin.QueryRowContext(ctx, `SELECT current_user`).Scan(&owner); err != nil {
		return err
	}
	for _, q := range []string{
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s, %s`, app, sys),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s, %s`, app, sys),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s, %s`, app, sys),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s, %s`, quoteIdent(owner), app, sys),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO %s, %s`, quoteIdent(owner), app, sys),
	} {
		if _, err := admin.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("%s: %w", q, err)
		}
	}
	return nil
}

func quoteIdent(s string) string {
	out := `"`
	for _, r := range s {
		if r == '"' {
			out += `"`
		}
		out += string(r)
	}
	return out + `"`
}

func quoteLiteral(s string) string {
	out := `'`
	for _, r := range s {
		if r == '\'' {
			out += `'`
		}
		out += string(r)
	}
	return out + `'`
}
