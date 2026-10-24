package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goStmtAllowed lista, por arquivo, as instruções `go` de produção que NÃO passam por safego. gin.Recovery só cobre o
// ciclo HTTP: um pânico numa goroutine solta derruba o processo inteiro. Cada entrada precisa de justificativa.
var goStmtAllowed = map[string]string{
	"pkg/safego/safego.go": "definição do próprio safego.Go",
}

// Toda goroutine de produção nasce por safego.Go (ou tem defer safego.Recover no topo). Rodar `go func(){...}()` direto
// é o que este teste impede.
func TestGoroutinesDeProducao_PassamPorSafego(t *testing.T) {
	root := businessRoot(t)
	fset := token.NewFileSet()
	bad := map[string]int{}

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
		rel = filepath.ToSlash(rel)
		ast.Inspect(f, func(n ast.Node) bool {
			g, ok := n.(*ast.GoStmt)
			if !ok {
				return true
			}
			if _, allowed := goStmtAllowed[rel]; allowed {
				return true
			}
			if goStmtRecovers(g) {
				return true
			}
			bad[rel]++
			return true
		})
		return nil
	})
	require.NoError(t, err)

	var files []string
	for f := range bad {
		files = append(files, f)
	}
	sort.Strings(files)
	assert.Empty(t, files, "goroutine sem contenção de pânico: use safego.Go (ou defer safego.Recover no topo)")
}

func TestGoStmtAllowed_TemJustificativa(t *testing.T) {
	for file, why := range goStmtAllowed {
		if len(strings.TrimSpace(why)) < 10 {
			t.Errorf("entrada %q sem justificativa: remova ou explique", file)
		}
	}
}

// goStmtRecovers diz se a goroutine `go func(){ defer safego.Recover(...) ... }()` contém o recover logo no início.
func goStmtRecovers(g *ast.GoStmt) bool {
	lit, ok := g.Call.Fun.(*ast.FuncLit)
	if !ok {
		return false
	}
	for _, stmt := range lit.Body.List {
		def, ok := stmt.(*ast.DeferStmt)
		if !ok {
			continue
		}
		if sel, ok := def.Call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Recover" {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "safego" {
				return true
			}
		}
	}
	return false
}
