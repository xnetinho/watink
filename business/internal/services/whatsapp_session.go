package services

import (
	"context"
	"fmt"
	"time"

	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/enginego"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultEngineType is used when a connection's EngineType is empty (legacy
// rows created before the column existed).
const defaultEngineType = "whatsmeow"

var _ domain.WhatsAppEngineResolver = (*WhatsAppSessionService)(nil)

// WhatsAppSessionService is the transport-agnostic session lifecycle
// dispatcher: it owns the start lock, status/broadcast bookkeeping shared by
// every engine, and resolves the concrete domain.WhatsAppEngine
// (whatsmeow/engine-go is the only one registered today) by Whatsapp.EngineType.
type WhatsAppSessionService struct {
	db        *gorm.DB
	publisher domain.CommandPublisher
	redisSvc  domain.RedisService
	broadcast domain.Broadcaster
	engines   map[string]domain.WhatsAppEngine
}

// NewWhatsAppSessionService wires the dispatcher via constructor DI. The
// whatsmeow/engine-go engine is always registered; extraEngines (optional)
// lets callers add further engines without breaking existing 4-arg call
// sites/tests.
func NewWhatsAppSessionService(db *gorm.DB, pub domain.CommandPublisher, redisSvc domain.RedisService, broadcast domain.Broadcaster, extraEngines ...map[string]domain.WhatsAppEngine) *WhatsAppSessionService {
	engines := map[string]domain.WhatsAppEngine{
		defaultEngineType: enginego.New(db, pub),
	}
	for _, m := range extraEngines {
		for k, v := range m {
			engines[k] = v
		}
	}
	return &WhatsAppSessionService{db: db, publisher: pub, redisSvc: redisSvc, broadcast: broadcast, engines: engines}
}

// EngineFor resolves the WhatsAppEngine for a connection's EngineType.
// Exported so callers outside session lifecycle (message send, FlowBuilder
// outbound) can resolve the same engine without depending on the full
// session service (see domain.WhatsAppEngineResolver).
func (wss *WhatsAppSessionService) EngineFor(whatsapp models.Whatsapp) (domain.WhatsAppEngine, error) {
	return wss.engineFor(whatsapp)
}

// engineFor resolves the WhatsAppEngine for a connection's EngineType.
func (wss *WhatsAppSessionService) engineFor(whatsapp models.Whatsapp) (domain.WhatsAppEngine, error) {
	et := whatsapp.EngineType
	if et == "" {
		et = defaultEngineType
	}
	e, ok := wss.engines[et]
	if !ok {
		return nil, fmt.Errorf("nenhum provedor registrado para engineType %q (conexão %d)", et, whatsapp.ID)
	}
	return e, nil
}

func (wss *WhatsAppSessionService) StartWhatsAppSession(whatsapp models.Whatsapp, usePairingCode bool, phoneNumber string, force bool) error {
	lockKey := fmt.Sprintf("session:start:%d", whatsapp.ID)
	lockValue := uuid.New().String()

	acquired, err := wss.redisSvc.SetLock(lockKey, lockValue, 10*time.Second)
	if err != nil {
		return err
	}
	if !acquired {
		// Clique duplo / outra aba: não é falha do servidor (500), é conflito esperado (409) — já
		// existe um start em andamento para esta conexão (lockKey TTL de 10s).
		return utils.NewFriendlyError(http.StatusConflict,
			"Esta conexão já está iniciando a sessão. Aguarde alguns segundos antes de tentar de novo.",
			fmt.Errorf("ERR_SESSION_STARTING_ALREADY"))
	}

	engine, err := wss.engineFor(whatsapp)
	if err != nil {
		_ = wss.redisSvc.DelLock(lockKey)
		return err
	}

	// RLS (ADR 0034): o handle global recusa consulta sem escopo de empresa. O serviço só recebe o modelo, então
	// abre o escopo a partir do tenant da própria conexão e o usa no UPDATE, no rollback e na resolução de proxy.
	ctx, done := tenantdb.TenantCtx(context.Background(), wss.db, whatsapp.TenantID)
	defer done()
	db := wss.db.WithContext(ctx)

	// previousStatus é o que a UI deve ver de volta se engine.StartSession falhar —
	// sem isso, a conexão fica travada em "OPENING" pra sempre (isBusy nunca
	// libera o botão "Gerar QR Code" no frontend).
	previousStatus := whatsapp.Status

	// Update Status — CRITICAL: tenant-scoped
	if err := db.Session(&gorm.Session{NewDB: true}).Model(&models.Whatsapp{}).
		Where("id = ? AND \"tenantId\" = ?", whatsapp.ID, whatsapp.TenantID).
		Update("status", "OPENING").Error; err != nil {
		_ = wss.redisSvc.DelLock(lockKey)
		return err
	}
	whatsapp.Status = "OPENING"

	// Emit via Socket (via injected broadcast)
	wss.broadcast.EmitToTenantRoom(whatsapp.TenantID.String(), "whatsappSession", map[string]interface{}{
		"action":  "update",
		"session": whatsapp,
	})
	wss.broadcast.EmitToTenantRoom(whatsapp.TenantID.String(), "channelConnection", map[string]interface{}{
		"action":     "update",
		"connection": whatsapp,
	})

	if err := engine.StartSession(ctx, whatsapp, usePairingCode, phoneNumber, force); err != nil {
		_ = wss.redisSvc.DelLock(lockKey)
		revertStatus := previousStatus
		if revertStatus == "" || revertStatus == "OPENING" || revertStatus == "PAIRING" {
			revertStatus = "DISCONNECTED"
		}
		if updErr := db.Session(&gorm.Session{NewDB: true}).Model(&models.Whatsapp{}).
			Where("id = ? AND \"tenantId\" = ?", whatsapp.ID, whatsapp.TenantID).
			Update("status", revertStatus).Error; updErr == nil {
			whatsapp.Status = revertStatus
			wss.broadcast.EmitToTenantRoom(whatsapp.TenantID.String(), "whatsappSession", map[string]interface{}{
				"action":  "update",
				"session": whatsapp,
			})
			wss.broadcast.EmitToTenantRoom(whatsapp.TenantID.String(), "channelConnection", map[string]interface{}{
				"action":     "update",
				"connection": whatsapp,
			})
		}
		return err
	}
	return nil
}

func (wss *WhatsAppSessionService) StopWhatsAppSession(whatsapp models.Whatsapp) error {
	engine, err := wss.engineFor(whatsapp)
	if err != nil {
		return err
	}
	return engine.StopSession(context.Background(), whatsapp)
}

func (wss *WhatsAppSessionService) DeleteWhatsAppSession(whatsapp models.Whatsapp) error {
	engine, err := wss.engineFor(whatsapp)
	if err != nil {
		return err
	}
	return engine.DeleteSession(context.Background(), whatsapp)
}
