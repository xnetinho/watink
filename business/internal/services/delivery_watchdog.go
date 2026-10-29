package services

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/safego"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"gorm.io/gorm"
)

const (
	// deliveryWatchdogInterval é a cadência da varredura.
	deliveryWatchdogInterval = 30 * time.Second
	// DeliveryTimeout: o engine confirma o envio (ack 1) em menos de 1 s. Parada em ack 0 por tanto tempo, a mensagem
	// nunca saiu: engine fora do ar, fila travada, comando perdido.
	DeliveryTimeout = 2 * time.Minute
	// deliveryWatchdogWindow limita a varredura às mensagens recentes: o que ficou parado há mais tempo já foi resolvido
	// (ou perdido) e varrer o histórico inteiro a cada ciclo seria desperdício.
	deliveryWatchdogWindow = 24 * time.Hour
	deliveryWatchdogLock   = "scheduler:cron:delivery-watchdog"
	ackPending             = 0 // gravada pelo business, o engine ainda não confirmou o envio
	ackFailed              = 5
)

type watchdogLock interface {
	SetLock(key string, value string, expiration time.Duration) (bool, error)
}

// DeliveryWatchdog marca como falha (ack 5) a mensagem que o engine nunca enviou: continua em ack 0 depois do prazo.
// É falha NOSSA (engine fora do ar, fila travada) e o operador precisa saber.
//
// Ack 1 NÃO é vigiado. Um tique só significa "o servidor do WhatsApp aceitou": com o celular do contato desligado ou sem
// internet a mensagem fica ali por dias e é entregue quando ele volta. Marcar isso como erro daria alarme falso e
// o reenvio duplicaria a mensagem. Pelo mesmo motivo ack 2 (entregue) não é falha: falta só o "lido".
//
// Não reenvia sozinho: o reenvio cria outro id de mensagem e pode duplicar; o operador decide pelo botão.
type DeliveryWatchdog struct {
	db        *gorm.DB
	redis     watchdogLock
	broadcast domain.Broadcaster
	timeout   time.Duration
	now       func() time.Time
}

func NewDeliveryWatchdog(db *gorm.DB, redis watchdogLock, broadcast domain.Broadcaster) *DeliveryWatchdog {
	return &DeliveryWatchdog{db: db, redis: redis, broadcast: broadcast, timeout: DeliveryTimeout, now: time.Now}
}

// Start roda o laço até o ctx ser cancelado. Chamar em goroutine própria no boot.
func (w *DeliveryWatchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(deliveryWatchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Um pânico num ciclo não pode matar o vigia para sempre: o próximo tick tenta de novo.
			_ = safego.Run("delivery.watchdog.tick", func() error { w.Tick(); return nil })
		}
	}
}

type stuckMessage struct {
	ID       string
	TicketID int
	TenantID string
}

// Tick faz uma varredura e devolve quantas mensagens marcou como falha.
func (w *DeliveryWatchdog) Tick() int {
	if w.redis != nil {
		ok, err := w.redis.SetLock(deliveryWatchdogLock, "1", deliveryWatchdogInterval-time.Second)
		if err != nil {
			log.Printf("[delivery.watchdog] lock: %v; pulando o ciclo", err)
			return 0
		}
		if !ok {
			return 0 // outro nó é o dono deste ciclo
		}
	}

	// Varredura de todas as empresas: caso de sistema (RLS, ADR 0034).
	sysCtx, done := tenantdb.SystemCtx(context.Background(), w.db)
	defer done()

	now := w.now()
	var stuck []stuckMessage
	// O UPDATE e a leitura são uma operação só (RETURNING): duas varreduras nunca marcam a mesma mensagem, e uma mensagem
	// que o recibo atualizou no meio do caminho (ack passou de 1) não casa mais com a condição.
	err := w.db.WithContext(sysCtx).Raw(
		`UPDATE "Messages" SET ack = ?, "updatedAt" = ?
		 WHERE "fromMe" = true AND ack = ? AND "mediaType" <> 'call' AND "isDeleted" = false
		   AND "createdAt" < ? AND "createdAt" > ?
		 RETURNING id AS id, "ticketId" AS ticket_id, "tenantId"::text AS tenant_id`,
		ackFailed, now, ackPending, now.Add(-w.timeout), now.Add(-deliveryWatchdogWindow),
	).Scan(&stuck).Error
	if err != nil {
		log.Printf("[delivery.watchdog] varredura falhou: %v", err)
		return 0
	}
	for _, m := range stuck {
		log.Printf("[delivery.watchdog] mensagem %s (ticket %d) sem confirmação de entrega em %s: marcada como falha", m.ID, m.TicketID, w.timeout)
		w.announce(sysCtx, m)
	}
	return len(stuck)
}

// announce avisa a tela para o tique de erro e o botão de reenviar aparecerem na hora.
func (w *DeliveryWatchdog) announce(ctx context.Context, m stuckMessage) {
	if w.broadcast == nil {
		return
	}
	var msg models.Message
	if err := w.db.WithContext(ctx).Where(`id = ?`, m.ID).Take(&msg).Error; err != nil {
		return
	}
	w.broadcast.EmitToRoom("/", "chat:"+strconv.Itoa(m.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": msg})
}
