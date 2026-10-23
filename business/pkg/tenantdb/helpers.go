package tenantdb

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Os helpers abaixo aceitam qualquer *gorm.DB. Sobre um handle que não é um Router (testes unitários com banco simples),
// passam direto: o isolamento por RLS só existe onde o Router existe.

// RouterOf devolve o Router por trás do handle, se houver.
func RouterOf(db *gorm.DB) (*Router, bool) {
	if db == nil {
		return nil, false
	}
	r, ok := db.ConnPool.(*Router)
	return r, ok
}

func noop() {}

// TenantCtx devolve um ctx com escopo da empresa e a função que o encerra. Chame o encerramento com defer.
func TenantCtx(ctx context.Context, db *gorm.DB, tenant uuid.UUID) (context.Context, func()) {
	r, ok := RouterOf(db)
	if !ok {
		return ctx, noop
	}
	s := r.NewScope(tenant)
	return WithScope(ctx, s), s.Close
}

// SystemCtx devolve um ctx com escopo de sistema (BYPASSRLS). Restrito por teste de arquitetura.
func SystemCtx(ctx context.Context, db *gorm.DB) (context.Context, func()) {
	r, ok := RouterOf(db)
	if !ok {
		return ctx, noop
	}
	s := r.SystemScope()
	return WithScope(ctx, s), s.Close
}

// WithTenant executa fn no escopo da empresa.
func WithTenant(ctx context.Context, db *gorm.DB, tenant uuid.UUID, fn func(ctx context.Context) error) error {
	c, done := TenantCtx(ctx, db, tenant)
	defer done()
	return fn(c)
}

// WithSystem executa fn no escopo de sistema.
func WithSystem(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	c, done := SystemCtx(ctx, db)
	defer done()
	return fn(c)
}

// Detach cria um escopo independente, do mesmo tipo do de ctx, para goroutines que sobrevivem à requisição ou ao evento
// que as criou. O ctx devolvido não herda cancelamento nem prazo; chame o encerramento com defer na goroutine.
func Detach(ctx context.Context) (context.Context, func()) {
	s := ScopeFrom(ctx)
	if s == nil {
		return context.WithoutCancel(ctx), noop
	}
	var ns *Scope
	if s.system {
		ns = s.r.SystemScope()
	} else {
		ns = s.r.NewScope(s.tenant)
	}
	return WithScope(context.WithoutCancel(ctx), ns), ns.Close
}

// Release devolve ao pool a conexão fixada do escopo de ctx. Chame antes de esperar uma chamada externa lenta (IA,
// geocodificação, sondagem) ou de abrir um fluxo longo; o próximo comando volta a fixar sozinho.
func Release(ctx context.Context) {
	if s := ScopeFrom(ctx); s != nil {
		s.Release()
	}
}

// Do executa fn com um handle no escopo da empresa e devolve a conexão ao pool em seguida. É o padrão para serviços que
// recebem só o tenantID e não têm um ponto natural de encerramento: o escopo nunca fica aberto depois de fn.
// O handle recebido por fn é novo (sem condições acumuladas).
func Do(ctx context.Context, db *gorm.DB, tenant uuid.UUID, fn func(tx *gorm.DB) error) error {
	c, done := TenantCtx(ctx, db, tenant)
	defer done()
	return fn(db.WithContext(c).Session(&gorm.Session{NewDB: true}))
}

// DoSystem é o equivalente de Do para o escopo de sistema.
func DoSystem(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	c, done := SystemCtx(ctx, db)
	defer done()
	return fn(db.WithContext(c).Session(&gorm.Session{NewDB: true}))
}
