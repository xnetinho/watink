package routes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Chamadas do frontend a rotas que não existem no backend e que JÁ eram assim antes da consolidação (verificado contra
// a base 8f214d7d). Cada uma tem motivo; entrada nova aqui exige justificativa. Remover a entrada quando a rota passar a existir (o teste cobra).
var frontendRouteDebt = map[string]string{
	"GET /tickets/:p/participants": "menção em grupo: o backend nunca expôs a rota",
	"GET /tags/:p":                 "TagModal busca a tag por id; só existe PUT/DELETE /tags/:id",
	"GET /v1/plugins/installed":    "NodesSidebar usa /v1 duplicado (baseURL já termina em /api/v1); a rota certa é /plugins/installed",
	"DELETE /messages/:p":          "apagar mensagem no chat: só existe GET/POST /messages/:ticketId",
	"POST /tickets/:p/history":     "registro de histórico no ticket: o backend só tem POST /tickets/:id/history/recover",
	"PUT /users/:p/configs":        "preferências do Dashboard: o backend nunca expôs a rota",
	"POST /auth/reset-password":    "tela de redefinir senha: o backend nunca implementou",
	"POST /saas/tenants/:p/plan":   "painel SaaS: as rotas vivem no control plane (Watink SaaS), não neste backend",
	"PUT /saas/plans/:p":           "painel SaaS: idem",
	"DELETE /saas/plans/:p":        "painel SaaS: idem",
	"GET /register/plans":          "registro self-service: a rota só é montada com SAAS_BASE_URL/SAAS_INSTANCE_ID/SAAS_INTERNAL_TOKEN",
	"GET /register/:p/status":      "registro self-service: idem",
	"POST /register":               "registro self-service: idem",
}

var (
	feCall  = regexp.MustCompile("api\\.(get|post|put|delete|patch)(?:<[^>]*>)?\\(\\s*[`\"'](/(?:\\$\\{[^}]*\\}|[^`\"'$])*)[`\"']")
	feParam = regexp.MustCompile(`\$\{[^}]*\}`)
)

// Contrato frontend → backend: toda chamada literal `api.<verbo>("/rota")` do frontend precisa casar com uma rota
// registrada em SetupRoutes (método e caminho). Pega a classe de bug "a tela chama uma rota que o backend não tem",
// como o `GET /connections/:id` que devolveu "connection not found" ao clicar em Ver QRCode.
func TestFrontendCallsExistOnBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")
	db := testutil.NewTestDB(t)
	r := gin.New()
	SetupRoutes(r.Group("/api/v1"), nil, application.NewContainer(db, nil, nil, nil), nil, nil, nil, nil)

	registered := map[string]bool{}
	for _, ri := range r.Routes() {
		path := strings.TrimPrefix(ri.Path, "/api/v1")
		registered[ri.Method+" "+normalizeRoute(path)] = true
	}

	src := filepath.Join("..", "..", "..", "frontend", "src")
	if _, err := os.Stat(src); err != nil {
		t.Skip("frontend/src não está disponível neste ambiente")
	}

	var missing []string
	seenDebt := map[string]bool{}
	require.NoError(t, filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !(strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx")) ||
			strings.Contains(p, "__tests__") || strings.Contains(p, ".test.") {
			return err
		}
		b, _ := os.ReadFile(p)
		for _, m := range feCall.FindAllStringSubmatch(string(b), -1) {
			path := feParam.ReplaceAllString(m[2], ":p")
			path = strings.TrimRight(strings.SplitN(path, "?", 2)[0], "/")
			key := strings.ToUpper(m[1]) + " " + normalizeRoute(path)
			if matchesRegistered(registered, strings.ToUpper(m[1]), path) {
				continue
			}
			if _, ok := frontendRouteDebt[key]; ok {
				seenDebt[key] = true
				continue
			}
			rel, _ := filepath.Rel(src, p)
			missing = append(missing, key+"  <- "+rel)
		}
		return nil
	}))

	assert.Empty(t, missing, "o frontend chama rota inexistente no backend (corrija a chamada, registre a rota ou, se for dívida antiga, documente em frontendRouteDebt)")
	for key := range frontendRouteDebt {
		assert.True(t, seenDebt[key], "dívida velha: %q não é mais chamada pelo frontend ou já existe no backend; remova de frontendRouteDebt", key)
	}
}

// matchesRegistered compara por segmento: um :p da chamada (valor dinâmico, ex.: /proxies/${id}/${acao}) casa com
// qualquer segmento da rota registrada; um :param da rota casa com qualquer valor da chamada.
func matchesRegistered(registered map[string]bool, method, call string) bool {
	cs := strings.Split(call, "/")
	for key := range registered {
		m, rpath, _ := strings.Cut(key, " ")
		if m != method {
			continue
		}
		rs := strings.Split(rpath, "/")
		if len(rs) != len(cs) {
			continue
		}
		ok := true
		for i := range rs {
			if rs[i] != cs[i] && rs[i] != ":p" && cs[i] != ":p" {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// normalizeRoute troca qualquer :nome por :p e tira a barra final, para comparar rota registrada com a chamada.
func normalizeRoute(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	for i, s := range parts {
		if strings.HasPrefix(s, ":") {
			parts[i] = ":p"
		}
	}
	return strings.Join(parts, "/")
}
