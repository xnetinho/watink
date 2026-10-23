// Package tenantdb impõe o isolamento entre empresas no banco (RLS) sem depender dos filtros da aplicação.
//
// O Router é o gorm.ConnPool do handle global: cada comando é roteado conforme o escopo que vem no ctx.
//
//   - ctx com escopo de empresa: o comando roda numa conexão fixada à empresa (app.tenant_id definido na sessão), e a
//     política RLS do banco só enxerga linhas dela;
//   - ctx com escopo de sistema: o comando roda no pool watink_system (BYPASSRLS), só para caminhos que não pertencem a
//     uma empresa (login, webhook, varreduras);
//   - ctx sem escopo: o comando roda no pool do app sem empresa. Tabela com dados de empresa é recusada com ErrNoScope
//     (falha ruidosa em vez de zero linhas em silêncio); o resto, como Tenants e Plans, funciona.
//
// Um Scope não é seguro para uso concorrente: ele fixa uma única conexão.
package tenantdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNoScope indica que um comando tocou tabela com dados de empresa sem escopo de empresa nem de sistema.
var ErrNoScope = errors.New("tenantdb: consulta sem escopo de empresa em tabela com dados de empresa")

type ctxKey struct{}

// ErrScopeClosed indica uso de um escopo depois do fim da requisição ou tarefa que o criou (por exemplo, uma goroutine
// que herdou o ctx da requisição). Falha ruidosa em vez de vazar uma conexão fixada.
var ErrScopeClosed = errors.New("tenantdb: escopo já encerrado (use Detach para goroutines)")

// Scope é o contexto de empresa (ou de sistema) de uma requisição, evento ou tarefa.
type Scope struct {
	r      *Router
	tenant uuid.UUID
	system bool

	mu     sync.Mutex
	conn   *sql.Conn
	closed bool
}

// WithScope devolve um ctx que carrega o escopo.
func WithScope(ctx context.Context, s *Scope) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// ScopeFrom devolve o escopo do ctx, ou nil.
func ScopeFrom(ctx context.Context) *Scope {
	s, _ := ctx.Value(ctxKey{}).(*Scope)
	return s
}

// TenantID devolve a empresa do escopo; ok=false para escopo de sistema ou ausente.
func (s *Scope) TenantID() (uuid.UUID, bool) {
	if s == nil || s.system {
		return uuid.Nil, false
	}
	return s.tenant, true
}

// IsSystem informa se o escopo é o de sistema.
func (s *Scope) IsSystem() bool { return s != nil && s.system }

// Release devolve a conexão fixada ao pool. Um novo comando volta a fixar sozinho. Use antes de esperar chamada externa
// ou de abrir um fluxo longo (SSE, WebSocket).
func (s *Scope) Release() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()
	if conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// A empresa nunca pode sobreviver na conexão devolvida: se não der para limpar, a conexão é descartada.
	if _, err := conn.ExecContext(ctx, `RESET app.tenant_id`); err != nil {
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	}
	_ = conn.Close()
}

// Close encerra o escopo: devolve a conexão e recusa qualquer uso posterior.
func (s *Scope) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.Release()
}

func (s *Scope) pinned(ctx context.Context) (*sql.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrScopeClosed
	}
	if s.conn != nil {
		return s.conn, nil
	}
	conn, err := s.r.app.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, `SELECT set_config('app.tenant_id', $1, false)`, s.tenant.String()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("tenantdb: definir empresa: %w", err)
	}
	s.conn = conn
	return conn, nil
}

// Router é o gorm.ConnPool que roteia por escopo.
type Router struct {
	app, sys *sql.DB
	tables   atomic.Pointer[map[string]struct{}]
}

// NewRouter cria o roteador. app é o pool do papel sem BYPASSRLS; sys é o pool de sistema (pode ser nil em testes que
// não usam escopo de sistema).
func NewRouter(app, sys *sql.DB) (*Router, error) {
	r := &Router{app: app, sys: sys}
	if err := r.RefreshCatalog(context.Background()); err != nil {
		return nil, err
	}
	return r, nil
}

// RefreshCatalog relê do catálogo as tabelas guardadas por política (com tenantId, filhas e a própria Tenants). Uma
// consulta sem escopo nelas é recusada com ErrNoScope, em vez de devolver zero linhas em silêncio.
func (r *Router) RefreshCatalog(ctx context.Context) error {
	db := r.sys
	if db == nil {
		db = r.app
	}
	cat, err := ReadCatalog(ctx, db)
	if err != nil {
		return fmt.Errorf("tenantdb: ler catálogo: %w", err)
	}
	set := map[string]struct{}{}
	for _, t := range cat.Guarded() {
		set[t] = struct{}{}
	}
	r.tables.Store(&set)
	return nil
}

// NewScope abre um escopo de empresa.
func (r *Router) NewScope(tenant uuid.UUID) *Scope { return &Scope{r: r, tenant: tenant} }

// SystemScope abre o escopo de sistema (BYPASSRLS). Restrito por teste de arquitetura aos pacotes autorizados.
func (r *Router) SystemScope() *Scope { return &Scope{r: r, system: true} }

// WithTenant executa fn com um escopo de empresa e o encerra no fim.
func (r *Router) WithTenant(ctx context.Context, tenant uuid.UUID, fn func(ctx context.Context) error) error {
	s := r.NewScope(tenant)
	defer s.Close()
	return fn(WithScope(ctx, s))
}

// WithSystem executa fn com o escopo de sistema.
func (r *Router) WithSystem(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(WithScope(ctx, r.SystemScope()))
}

var identRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

func (r *Router) touchesTenantTable(query string) bool {
	set := r.tables.Load()
	if set == nil {
		return false
	}
	for _, tok := range identRe.FindAllString(query, -1) {
		if _, ok := (*set)[tok]; ok {
			return true
		}
	}
	return false
}

type executor interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *Router) executor(ctx context.Context, query string) (executor, error) {
	switch s := ScopeFrom(ctx); {
	case s == nil:
		if r.touchesTenantTable(query) {
			logNoScope(query)
			return nil, ErrNoScope
		}
		return r.app, nil
	case s.system:
		if r.sys == nil {
			return nil, errors.New("tenantdb: pool de sistema não configurado")
		}
		return r.sys, nil
	default:
		return s.pinned(ctx)
	}
}

// logNoScope registra onde a consulta sem escopo nasceu: o primeiro frame do próprio projeto fora deste pacote.
func logNoScope(query string) {
	where := "?"
	pcs := make([]uintptr, 24)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if strings.Contains(f.File, "/business/") && !strings.Contains(f.File, "/pkg/tenantdb/") {
			where = fmt.Sprintf("%s:%d", f.File[strings.Index(f.File, "/business/")+10:], f.Line)
			break
		}
		if !more {
			break
		}
	}
	q := strings.ReplaceAll(query, "\n", " ")
	if len(q) > 110 {
		q = q[:110]
	}
	log.Printf("[tenantdb] SEM ESCOPO em %s: %s", where, q)
}

func (r *Router) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	ex, err := r.executor(ctx, query)
	if err != nil {
		return nil, err
	}
	return ex.PrepareContext(ctx, query)
}

func (r *Router) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ex, err := r.executor(ctx, query)
	if err != nil {
		return nil, err
	}
	return ex.ExecContext(ctx, query, args...)
}

func (r *Router) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ex, err := r.executor(ctx, query)
	if err != nil {
		return nil, err
	}
	return ex.QueryContext(ctx, query, args...)
}

func (r *Router) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ex, err := r.executor(ctx, query)
	if err != nil {
		// sql.Row não carrega erro próprio: um ctx cancelado faz o Scan devolver erro.
		dead, cancel := context.WithCancel(ctx)
		cancel()
		return r.app.QueryRowContext(dead, query)
	}
	return ex.QueryRowContext(ctx, query, args...)
}

// BeginTx implementa gorm.ConnPoolBeginner: a transação nasce na conexão do escopo.
func (r *Router) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	switch s := ScopeFrom(ctx); {
	case s == nil:
		tx, err := r.app.BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &strictTx{Tx: tx, r: r}, nil
	case s.system:
		if r.sys == nil {
			return nil, errors.New("tenantdb: pool de sistema não configurado")
		}
		return r.sys.BeginTx(ctx, opts)
	default:
		conn, err := s.pinned(ctx)
		if err != nil {
			return nil, err
		}
		return conn.BeginTx(ctx, opts)
	}
}

// strictTx mantém a recusa de tabela de empresa dentro de transação aberta sem escopo.
type strictTx struct {
	*sql.Tx
	r *Router
}

func (t *strictTx) guard(query string) error {
	if t.r.touchesTenantTable(query) {
		logNoScope(query)
		return ErrNoScope
	}
	return nil
}

func (t *strictTx) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	if err := t.guard(q); err != nil {
		return nil, err
	}
	return t.Tx.PrepareContext(ctx, q)
}

func (t *strictTx) ExecContext(ctx context.Context, q string, a ...any) (sql.Result, error) {
	if err := t.guard(q); err != nil {
		return nil, err
	}
	return t.Tx.ExecContext(ctx, q, a...)
}

func (t *strictTx) QueryContext(ctx context.Context, q string, a ...any) (*sql.Rows, error) {
	if err := t.guard(q); err != nil {
		return nil, err
	}
	return t.Tx.QueryContext(ctx, q, a...)
}

func (t *strictTx) QueryRowContext(ctx context.Context, q string, a ...any) *sql.Row {
	if err := t.guard(q); err != nil {
		dead, cancel := context.WithCancel(ctx)
		cancel()
		return t.Tx.QueryRowContext(dead, q)
	}
	return t.Tx.QueryRowContext(ctx, q, a...)
}

// Ping e GetDBConn mantêm o contrato esperado pelo GORM (gorm.Open e db.DB()).
func (r *Router) Ping() error { return r.app.Ping() }

func (r *Router) GetDBConn() (*sql.DB, error) { return r.app, nil }
