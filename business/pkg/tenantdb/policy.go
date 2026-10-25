package tenantdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// tenantExpr é o valor de empresa da sessão. Sem empresa definida o resultado é NULL, e comparação com NULL nunca é
// verdadeira: o banco falha fechado.
const tenantExpr = `NULLIF(current_setting('app.tenant_id', true), '')::uuid`

// Child descreve uma tabela sem coluna de empresa que herda o isolamento de uma tabela pai (que tem política).
type Child struct {
	FK       string // coluna desta tabela que aponta para o pai
	Parent   string // tabela pai
	ParentPK string // coluna chave do pai
}

// Children lista as tabelas filhas e de junção sem tenantId. A linha só é visível se o pai for visível: o EXISTS roda
// sob a política do pai, então empresa A nunca enxerga filho de empresa B.
var Children = map[string]Child{
	"PipelineStages":            {"pipelineId", "Pipelines", "id"},
	"ProductSKUs":               {"productId", "Products", "id"},
	"SKUPrices":                 {"skuId", "ProductSKUs", "id"},
	"ProductCompositions":       {"parentSkuId", "ProductSKUs", "id"},
	"WarehouseBalances":         {"warehouseId", "Warehouses", "id"},
	"ProtocolAttachments":       {"protocolId", "Protocols", "id"},
	"PollResults":               {"pollMessageId", "Messages", "id"},
	"cargo_permissoes":          {"cargoId", "Cargos", "id"},
	"setor_filas":               {"setorId", "Setores", "id"},
	"user_setores":              {"userId", "Users", "id"},
	"user_queues":               {"user_id", "Users", "id"},
	"channel_connection_queues": {"channel_connection_id", "ChannelConnections", "id"},
	"whatsapp_queues":           {"whatsapp_id", "Whatsapps", "id"},
	"whatsapp_connection_details": {"connectionId", "ChannelConnections", "id"},
}

// SelfPolicies são tabelas em que a própria chave identifica a empresa.
var SelfPolicies = map[string]string{
	"Tenants": "id",
}

// Exempt lista as tabelas realmente globais (sem dono de empresa). Toda tabela fora de Children, SelfPolicies e sem
// tenantId precisa estar aqui, senão o teste de catálogo falha.
var Exempt = map[string]string{
	"Plans":            "catálogo de planos da instalação",
	"Permissions":      "catálogo de permissões da instalação",
	"InstancePolicies": "política da instalação definida pelo SaaS",
	"SchemaMigrations": "controle de versão de migrações globais do banco",
	"spatial_ref_sys":  "tabela do PostGIS",
}

// exemptPrefixes isenta famílias de tabelas de bibliotecas (armazenamento do whatsmeow).
var exemptPrefixes = []string{"whatsmeow_"}

func isExempt(table string) bool {
	if _, ok := Exempt[table]; ok {
		return true
	}
	for _, p := range exemptPrefixes {
		if strings.HasPrefix(table, p) {
			return true
		}
	}
	return false
}

func quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// Catalog descreve o que o catálogo do banco contém, para gerar e verificar políticas.
type Catalog struct {
	WithTenant []string // tabelas com coluna tenantId
	Others     []string // tabelas sem tenantId
}

// ReadCatalog lê as tabelas base do schema corrente.
func ReadCatalog(ctx context.Context, db *sql.DB) (Catalog, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.table_name,
		       EXISTS (SELECT 1 FROM information_schema.columns c
		               WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name AND c.column_name = 'tenantId')
		FROM information_schema.tables t
		WHERE t.table_type = 'BASE TABLE' AND t.table_schema = current_schema()
		ORDER BY t.table_name`)
	if err != nil {
		return Catalog{}, err
	}
	defer func() { _ = rows.Close() }()
	var cat Catalog
	for rows.Next() {
		var name string
		var has bool
		if err := rows.Scan(&name, &has); err != nil {
			return Catalog{}, err
		}
		if has {
			cat.WithTenant = append(cat.WithTenant, name)
		} else {
			cat.Others = append(cat.Others, name)
		}
	}
	return cat, rows.Err()
}

// Unclassified devolve as tabelas sem tenantId que não estão em Children, SelfPolicies nem Exempt.
func (c Catalog) Unclassified() []string {
	var out []string
	for _, t := range c.Others {
		if _, ok := Children[t]; ok {
			continue
		}
		if _, ok := SelfPolicies[t]; ok {
			continue
		}
		if isExempt(t) {
			continue
		}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// Guarded devolve todas as tabelas com política, na ordem do catálogo.
func (c Catalog) Guarded() []string {
	set := map[string]struct{}{}
	for _, t := range c.WithTenant {
		set[t] = struct{}{}
	}
	for _, t := range c.Others {
		if _, ok := Children[t]; ok {
			set[t] = struct{}{}
		}
		if _, ok := SelfPolicies[t]; ok {
			set[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func policyName(table string) string { return "tenant_isolation" }

func usingExpr(table string, cat Catalog) string {
	if col, ok := SelfPolicies[table]; ok {
		return fmt.Sprintf(`%s = %s`, quote(col), tenantExpr)
	}
	if ch, ok := Children[table]; ok {
		return fmt.Sprintf(`EXISTS (SELECT 1 FROM %s p WHERE p.%s = %s.%s)`,
			quote(ch.Parent), quote(ch.ParentPK), quote(table), quote(ch.FK))
	}
	return fmt.Sprintf(`%s = %s`, quote("tenantId"), tenantExpr)
}

// clusterLockKey identifica o lock de catálogo do cluster.
const clusterLockKey = 727275

// LockCluster serializa tudo o que altera catálogos COMPARTILHADOS DO CLUSTER (papéis, privilégios, dependências de
// papéis em CREATE/DROP DATABASE). Duas operações simultâneas falham com "tuple concurrently updated".
//
// ATENÇÃO: um advisory lock do Postgres vale DENTRO DO BANCO em que foi tomado. Dois bancos diferentes não se bloqueiam,
// embora compartilhem os papéis. Por isso o lock é tomado numa conexão ao banco de manutenção "postgres", comum a todas
// as migrações do cluster, e não no banco que está sendo migrado. Quem chama passa um *sql.DB já conectado a ele ou
// usa LockClusterVia, que abre a conexão a partir do DSN do banco em migração.
func LockCluster(ctx context.Context, db *sql.DB) (unlock func(), err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var dbname string
	if err := conn.QueryRowContext(ctx, `SELECT current_database()`).Scan(&dbname); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if dbname != "postgres" {
		_ = conn.Close()
		return nil, fmt.Errorf("tenantdb: LockCluster exige conexão ao banco de manutenção %q (veio %q); use LockClusterVia", "postgres", dbname)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, clusterLockKey); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("tenantdb: lock de catálogo: %w", err)
	}
	return func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, clusterLockKey)
		_ = conn.Close()
	}, nil
}

// LockClusterVia abre uma conexão ao banco de manutenção "postgres" com as mesmas credenciais e endereço de db e toma
// o lock do cluster nela. É o que o job de migração usa, pois está conectado ao banco da aplicação.
func LockClusterVia(ctx context.Context, open func(dbname string) (*sql.DB, error)) (unlock func(), err error) {
	adm, err := open("postgres")
	if err != nil {
		return nil, err
	}
	un, err := LockCluster(ctx, adm)
	if err != nil {
		_ = adm.Close()
		return nil, err
	}
	return func() {
		un()
		_ = adm.Close()
	}, nil
}

// ApplyPolicies liga o RLS e recria a política única de cada tabela guardada. É idempotente. Deve rodar como dono dos
// objetos (papel de migração), nunca como o papel do aplicativo.
func ApplyPolicies(ctx context.Context, db *sql.DB) error {
	return ApplyPoliciesLocked(ctx, db)
}

// ApplyPoliciesLocked é ApplyPolicies para quem JÁ segura o lock do cluster (LockCluster): não toma o lock de novo, o que
// bloquearia esperando a si mesmo numa conexão diferente.
func ApplyPoliciesLocked(ctx context.Context, db *sql.DB) error {
	cat, err := ReadCatalog(ctx, db)
	if err != nil {
		return err
	}
	if bad := cat.Unclassified(); len(bad) > 0 {
		return fmt.Errorf("tenantdb: tabelas sem classificação de isolamento: %s", strings.Join(bad, ", "))
	}
	for _, t := range cat.Guarded() {
		stmts := []string{
			fmt.Sprintf(`ALTER TABLE %s ENABLE ROW LEVEL SECURITY`, quote(t)),
			fmt.Sprintf(`DROP POLICY IF EXISTS %s ON %s`, quote(policyName(t)), quote(t)),
		}
		expr := usingExpr(t, cat)
		stmts = append(stmts, fmt.Sprintf(`CREATE POLICY %s ON %s USING (%s) WITH CHECK (%s)`,
			quote(policyName(t)), quote(t), expr, expr))
		for _, s := range stmts {
			if _, err := db.ExecContext(ctx, s); err != nil {
				return fmt.Errorf("tenantdb: %s: %w", s, err)
			}
		}
	}
	return nil
}

// Verify devolve as tabelas guardadas sem RLS ativo ou sem a política, e as sem classificação.
func Verify(ctx context.Context, db *sql.DB) (problems []string, err error) {
	cat, err := ReadCatalog(ctx, db)
	if err != nil {
		return nil, err
	}
	for _, t := range cat.Unclassified() {
		problems = append(problems, "sem classificação: "+t)
	}
	rows, err := db.QueryContext(ctx, `
		SELECT c.relname, c.relrowsecurity, EXISTS (SELECT 1 FROM pg_policies p WHERE p.tablename = c.relname AND p.schemaname = n.nspname)
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r' AND n.nspname = current_schema()`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	state := map[string][2]bool{}
	for rows.Next() {
		var name string
		var rls, pol bool
		if err := rows.Scan(&name, &rls, &pol); err != nil {
			return nil, err
		}
		state[name] = [2]bool{rls, pol}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, t := range cat.Guarded() {
		s := state[t]
		if !s[0] {
			problems = append(problems, "RLS desligado: "+t)
		}
		if !s[1] {
			problems = append(problems, "sem política: "+t)
		}
	}
	return problems, nil
}
