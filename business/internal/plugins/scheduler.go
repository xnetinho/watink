package plugins

import (
	"context"
	"fmt"
	"github.com/alltomatos/watinkdev/business/pkg/safego"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
)

// cronScheduler implements the periodic-job half of sdk.WatinkCoreScheduler
// (ADR 0027). Each registered job runs on its own ticker; when redis is
// configured, a SetNX-style leader-lock (same primitive the FlowBuilder
// scheduler uses) ensures only one node in a multi-node deployment executes a
// given job name per tick. Without redis (e.g. NewPluginManager in tests),
// jobs still run — single-node semantics, no lock.
type cronScheduler struct {
	mu    sync.Mutex
	names map[string]bool
	redis domain.RedisService
	db    *gorm.DB // para abrir o escopo de sistema de cada tick (RLS, ADR 0034); nil nos testes sem banco
}

func newCronScheduler(redis domain.RedisService, db *gorm.DB) *cronScheduler {
	return &cronScheduler{names: make(map[string]bool), redis: redis, db: db}
}

func (s *cronScheduler) register(name string, interval time.Duration, fn func(ctx context.Context) error) error {
	if name == "" {
		return fmt.Errorf("cronScheduler: job name cannot be empty")
	}
	if interval <= 0 {
		return fmt.Errorf("cronScheduler: interval must be positive")
	}
	s.mu.Lock()
	if s.names[name] {
		s.mu.Unlock()
		return fmt.Errorf("cronScheduler: job %q already registered", name)
	}
	s.names[name] = true
	s.mu.Unlock()

	safego.Go("cron."+name, func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			// Um pânico num tick não pode matar o agendador do job: contido por tick.
			func() {
				defer safego.Recover("cron." + name)
				s.runOnce(name, interval, fn)
			}()
		}
	})
	return nil
}

// runOnce guards a single tick with the leader-lock (TTL just under the tick
// interval, so a missed DelLock never wedges the job past its own next tick)
// and runs fn with a context bounded by the tick interval.
func (s *cronScheduler) runOnce(name string, interval time.Duration, fn func(ctx context.Context) error) {
	lockKey := "scheduler:cron:" + name
	if s.redis != nil {
		ttl := interval - time.Second
		if ttl <= 0 {
			ttl = interval
		}
		acquired, err := s.redis.SetLock(lockKey, "1", ttl)
		if err != nil {
			log.Printf("[cronScheduler] lock error for job %q: %v — skipping this tick", name, err)
			return
		}
		if !acquired {
			return // another node already owns this tick
		}
	}
	// Todo cron de plugin varre várias empresas, então o tick roda no escopo de sistema; quem trata cada empresa em
	// seguida troca para o escopo dela.
	ctx, cancel := context.WithTimeout(context.Background(), interval)
	defer cancel()
	ctx, done := tenantdb.SystemCtx(ctx, s.db)
	defer done()
	if err := fn(ctx); err != nil {
		log.Printf("[cronScheduler] job %q failed: %v", name, err)
	}
}

// domainEventBus implements the Subscribe half of sdk.WatinkCoreScheduler —
// a minimal in-process pub/sub for internal domain events (e.g. Pipeline
// stage changes, see docs/agents/assistants.md § Eventos de Pipeline).
// Delivery is best-effort, no persistence/replay; publishers live in core
// code (e.g. DealController) and call PublishDomainEvent directly — they are
// not plugins, so they don't go through the sdk.WatinkCore interface.
type domainEventBus struct {
	mu   sync.RWMutex
	subs map[string][]func(ctx context.Context, payload map[string]any)
	db   *gorm.DB // abre o escopo de empresa de cada entrega (RLS, ADR 0034); nil nos testes sem banco
}

var eventBus = &domainEventBus{subs: make(map[string][]func(ctx context.Context, payload map[string]any))}

// SetEventBusDB liga o barramento ao banco para que cada entrega rode no escopo da empresa do evento.
func SetEventBusDB(db *gorm.DB) {
	eventBus.mu.Lock()
	eventBus.db = db
	eventBus.mu.Unlock()
}

func (b *domainEventBus) subscribe(eventType string, fn func(ctx context.Context, payload map[string]any)) error {
	if eventType == "" {
		return fmt.Errorf("domainEventBus: eventType cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("domainEventBus: fn cannot be nil")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[eventType] = append(b.subs[eventType], fn)
	return nil
}

func (b *domainEventBus) publish(ctx context.Context, eventType string, payload map[string]any) {
	b.mu.RLock()
	handlers := append([]func(ctx context.Context, payload map[string]any){}, b.subs[eventType]...)
	b.mu.RUnlock()
	b.mu.RLock()
	db := b.db
	b.mu.RUnlock()
	tenantID, hasTenant := uuid.Parse(fmt.Sprint(payload["tenantId"]))
	for _, fn := range handlers {
		// Cada assinante recebe um ctx com o escopo da empresa do evento: o publicador pode ser uma requisição, um
		// listener ou um cron, e os assinantes usam o *gorm.DB capturado no registro, que só enxerga a empresa certa
		// se o ctx a carregar (RLS, ADR 0034).
		hctx, done := ctx, func() {}
		if db != nil && hasTenant == nil {
			hctx, done = tenantdb.TenantCtx(context.WithoutCancel(ctx), db, tenantID)
		}
		func() {
			defer done()
			defer safego.Recover("domainEvent." + eventType)
			fn(hctx, payload)
		}()
	}
}

// PublishDomainEvent is the entry point for core code (outside the plugins
// package, e.g. controllers.DealController) to publish an internal domain
// event to any plugin subscribed via sdk.WatinkCoreScheduler.Subscribe.
func PublishDomainEvent(ctx context.Context, eventType string, payload map[string]any) {
	eventBus.publish(ctx, eventType, payload)
}
