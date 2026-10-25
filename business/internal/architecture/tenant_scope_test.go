// Package architecture guarda regras de arquitetura que o compilador não impõe.
package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// systemScopeAllowed lista, por arquivo (relativo a business/), quem pode abrir o escopo de sistema do banco (BYPASSRLS).
// Cada entrada é um caminho que NÃO pertence a uma empresa. Adicionar uma linha aqui é uma decisão de segurança e deve
// passar por revisão (ADR 0034, D6).
var systemScopeAllowed = map[string]string{
	"internal/controllers/channel_webhook.go":    "descobrir a empresa da conexão do webhook",
	"internal/knowledge/reconciler.go":           "varredura global de fontes travadas",
	"internal/services/event_listener.go":        "engine.ready enumera tenants globalmente antes de abrir escopo individual; não é dado de uma empresa",
	"internal/middleware/system_scope.go":        "rotas públicas e internas sem empresa (login, setup, SaaS)",
	"internal/plugins/manager.go":                "rotas públicas de plugin (empresa descoberta no handler)",
	"internal/plugins/scheduler.go":              "tick de cron que varre empresas",
	"internal/routes/routes.go":                  "retomada de polling no boot",
	"internal/saasclient/worker.go":              "sincronismo SaaS da instalação",
	"internal/services/saas_contract_service.go": "contrato SaaS é da instalação",
	"internal/services/setup_service.go":         "criar/alterar empresas e checar setup",
	"internal/services/tenant_status_service.go": "status da empresa no gate (antes do escopo)",
	"pkg/tenantdb/helpers.go":                    "definição do helper",
	"pkg/tenantdb/tenantdb.go":                   "definição do escopo",
}

func businessRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if dir == filepath.Dir(dir) {
			t.Fatal("go.mod não encontrado")
		}
	}
}

// callers devolve os arquivos de produção (não-teste) que chamam alguma das funções nomeadas.
func callers(t *testing.T, root string, names ...string) map[string][]string {
	t.Helper()
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	out := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "node_modules" || n == "build" || n == "docs" || n == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			var name string
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if want[name] {
				out[rel] = append(out[rel], name)
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	return out
}

func TestEscopoDeSistema_SoPacotesAutorizados(t *testing.T) {
	root := businessRoot(t)
	found := callers(t, root, "SystemCtx", "SystemScope", "WithSystem", "DoSystem")

	var violations []string
	for file := range found {
		if _, ok := systemScopeAllowed[filepath.ToSlash(file)]; !ok {
			violations = append(violations, file)
		}
	}
	sort.Strings(violations)
	assert.Empty(t, violations,
		"estes arquivos usam o escopo de sistema (BYPASSRLS) sem autorização; se o caminho realmente não pertence a uma "+
			"empresa, inclua-o em systemScopeAllowed com a justificativa (ADR 0034, D6)")
}

func TestEscopoDeSistema_ListaNaoTemEntradasMortas(t *testing.T) {
	root := businessRoot(t)
	found := callers(t, root, "SystemCtx", "SystemScope", "WithSystem", "DoSystem")
	var dead []string
	for file, why := range systemScopeAllowed {
		if why == "" {
			continue
		}
		if _, ok := found[file]; !ok {
			dead = append(dead, file)
		}
	}
	sort.Strings(dead)
	assert.Empty(t, dead, "entradas autorizadas que já não usam o escopo de sistema devem sair da lista")
}

func TestPoolDeSistema_NaoVazaParaForaDoPacoteDeAcesso(t *testing.T) {
	root := businessRoot(t)
	// RoleSystem/DB_SYSTEM_PASS só existem no pacote database; nenhum outro código pode abrir conexão como sistema.
	bad := []string{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/database/") || strings.HasPrefix(rel, "internal/architecture/") {
			return nil
		}
		b, _ := os.ReadFile(path)
		if strings.Contains(string(b), "DB_SYSTEM_PASS") || strings.Contains(string(b), "DB_ADMIN_PASS") {
			bad = append(bad, rel)
		}
		return nil
	})
	assert.Empty(t, bad, "credenciais de sistema/administrador só podem ser lidas em internal/database")
}
