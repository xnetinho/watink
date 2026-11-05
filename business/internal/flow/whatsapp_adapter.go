package flow

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	wbot "github.com/alltomatos/watinkdev/contracts/wbot"

	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

// dedupTTL is the idempotency window for an outbound send (mirrors the
// wbot:msg: Redis convention used across the message pipeline).
const dedupTTL = 24 * time.Hour

// WhatsAppAdapterDeps are optional dependencies enabling non-AMQP engines.
// When absent, every send goes through the AMQP path below —
// preserving the original FASE 1 behavior unchanged.
type WhatsAppAdapterDeps struct {
	DB      *gorm.DB
	Engines domain.WhatsAppEngineResolver
}

// WhatsAppAdapter is the OutboundChannelAdapter for the "whatsapp" channel. It
// dedups by EnvID, then either translates an OutboundMessage into the AMQP
// command `wbot.<tenant>.<session>.<cmd>` for the dumb engine-go executor
// (default, ADR 0014), or — when the connection's EngineType is non-AMQP
// and WhatsAppAdapterDeps is wired — dispatches via domain.WhatsAppEngine
// instead.
type WhatsAppAdapter struct {
	rabbit domain.CommandPublisher
	redis  domain.RedisService
	deps   WhatsAppAdapterDeps
}

// NewWhatsAppAdapter wires the adapter via constructor DI (no global/singleton).
// deps is optional (variadic) so existing call sites keep working unchanged.
func NewWhatsAppAdapter(rabbit domain.CommandPublisher, redis domain.RedisService, deps ...WhatsAppAdapterDeps) *WhatsAppAdapter {
	a := &WhatsAppAdapter{rabbit: rabbit, redis: redis}
	if len(deps) > 0 {
		a.deps = deps[0]
	}
	return a
}

// Channel identifies this adapter in the registry.
func (a *WhatsAppAdapter) Channel() string { return "whatsapp" }

// Send dedups by EnvID, then publishes the engine command. A duplicate (lock
// already held) is a no-op success — the redelivery must not double-send.
//
// Meta keys honored: "messageId" (string, the WhatsApp msg id; generated if
// absent — but the caller should supply the EnvID-derived id for the persisted
// row to match the ack), "mediaType"/"mediaUrl"/"mimeType" (string).
func (a *WhatsAppAdapter) Send(ctx context.Context, msg OutboundMessage) error {
	_, err := a.sendInternal(ctx, msg)
	return err
}

// SendReportingID behaves exactly like Send, but additionally returns the
// EFFECTIVE WhatsApp message ID actually assigned to the outbound message.
// For every AMQP/whatsmeow path this always equals the id we requested
// (engine-go forces it via whatsmeow's SendRequestExtra) -- but a
// non-AMQP domain.WhatsAppEngine that assigns its own id can return a
// DIFFERENT one, and callers that persist a Message row keyed by their own
// EnvID (campaign sendOne/dispatchCampaignMessage) need the real value, or
// reply correlation (QuotedMsgID matching) can never find it. Plain Send
// callers that don't care about this (FlowBuilder text/media nodes) keep
// using Send unchanged.
func (a *WhatsAppAdapter) SendReportingID(ctx context.Context, msg OutboundMessage) (string, error) {
	return a.sendInternal(ctx, msg)
}

func (a *WhatsAppAdapter) sendInternal(ctx context.Context, msg OutboundMessage) (string, error) {
	if msg.To == "" {
		return "", fmt.Errorf("whatsapp adapter: empty destination (to)")
	}
	if msg.SessionID == "" {
		return "", fmt.Errorf("whatsapp adapter: empty sessionId")
	}

	// Dedup BEFORE any real send. SetLock is SetNX: false => already sent.
	if a.redis != nil && msg.EnvID != "" {
		acquired, err := a.redis.SetLock("wbot:msg:"+msg.EnvID, "1", dedupTTL)
		if err != nil {
			return "", fmt.Errorf("whatsapp adapter: dedup lock: %w", err)
		}
		if !acquired {
			return msg.EnvID, nil
		}
	}

	// Resolve a conexão (ChannelConnections) e o canal cedo: a publicação final e o "digitando..." passam por ele. Só quando
	// as deps são injetadas; call sites que não as injetam publicam direto, como no FASE 1.
	var conn models.ChannelConnection
	var sender domain.RichSender
	var whatsapp models.Whatsapp
	var engine domain.WhatsAppEngine
	if a.deps.DB != nil {
		sid, _ := strconv.Atoi(msg.SessionID)
		if err := a.deps.DB.Session(&gorm.Session{NewDB: true}).
			Where(`id = ? AND "tenantId" = ?`, sid, msg.TenantID).First(&conn).Error; err == nil {
			if ch, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind)); err == nil {
				sender, _ = ch.(domain.RichSender)
			}
		}
		if a.deps.Engines != nil {
			if err := models.FirstWhatsapp(a.deps.DB.Session(&gorm.Session{NewDB: true}).
				Where(`id = ? AND "tenantId" = ?`, sid, msg.TenantID), &whatsapp); err == nil {
				engine, _ = a.deps.Engines.EngineFor(whatsapp)
			}
		}
	}
	a.applyHumanPacing(ctx, engine, whatsapp, msg)

	// Rich sends (interactive/media/poll/carousel — e.g. the quickAnswer node)
	// carry a pre-built commandType+payload in Meta instead of the plain
	// body/mediaUrl shape below. BuildQuickAnswerCommand already fills
	// sessionId/messageId/to, so pass it through verbatim.
	if ct, ok := msg.Meta["commandType"].(string); ok && ct != "" {
		payload, _ := msg.Meta["payload"].(map[string]interface{})
		if err := a.publish(ctx, sender, &conn, msg, ct, payload); err != nil {
			if a.redis != nil && msg.EnvID != "" {
				if delErr := a.redis.DelLock("wbot:msg:" + msg.EnvID); delErr != nil {
					log.Printf("[WhatsAppAdapter] dedup lock release after publish failure failed (env=%s): %v", msg.EnvID, delErr)
				}
			}
			return "", fmt.Errorf("whatsapp adapter: publish rich command: %w", err)
		}
		effectiveID, _ := payload["messageId"].(string)
		if effectiveID == "" {
			effectiveID = msg.EnvID
		}
		return effectiveID, nil
	}

	mediaType := metaString(msg.Meta, "mediaType")
	mediaURL := metaString(msg.Meta, "mediaUrl")
	mimeType := metaString(msg.Meta, "mimeType")
	// mediaData: bytes crus em base64 — usado pelo Assistant Runtime pra
	// enviar áudio sintetizado (SendAssistantMedia) sem precisar de uma URL
	// pública alcançável pelo engine-go (o mesmo host já publica a mensagem
	// AMQP; engine-go decodifica direto, ver resolveMediaBytes). mediaUrl
	// continua o caminho normal do resto do FlowBuilder (nó "message" etc.).
	mediaData := metaString(msg.Meta, "mediaData")

	commandType := "message.send.text"
	if mediaURL != "" || mediaData != "" {
		commandType = "message.send.media"
	}

	messageID := metaString(msg.Meta, "messageId")
	if messageID == "" {
		messageID = msg.EnvID
	}

	// The engine unmarshals payload.sessionId into an `int` (no `,string` tag —
	// see engine-go send_types.go). Sending it as a string makes that unmarshal
	// fail → the command is NACKed and the message is NEVER sent. Convert here;
	// the routing-key segment stays a string (AMQP keys are textual). A
	// non-numeric SessionID yields 0, which the engine rejects loudly rather than
	// silently mis-routing.
	sid, _ := strconv.Atoi(msg.SessionID)

	payload := map[string]interface{}{
		"sessionId": sid,
		"messageId": messageID,
		"to":        msg.To,
		"body":      msg.Body,
		"mediaType": mediaType,
		"mediaUrl":  mediaURL,
		"mediaData": mediaData,
		"mimeType":  mimeType,
	}
	if err := a.publish(ctx, sender, &conn, msg, commandType, payload); err != nil {
		// Publish failed AFTER we took the dedup lock. If we leave the lock held,
		// the AMQP redelivery (retry of this same outbound) hits the retained lock
		// and no-ops silently → the message is lost for the full 24h TTL. Release
		// the lock best-effort so the retry can actually re-send. Only release the
		// lock we just acquired here (msg.EnvID != "" means we took it above).
		if a.redis != nil && msg.EnvID != "" {
			if delErr := a.redis.DelLock("wbot:msg:" + msg.EnvID); delErr != nil {
				log.Printf("[WhatsAppAdapter] dedup lock release after publish failure failed (env=%s): %v", msg.EnvID, delErr)
			}
		}
		return "", fmt.Errorf("whatsapp adapter: publish command: %w", err)
	}
	return messageID, nil
}

// applyHumanPacing shows the "digitando..." indicator and waits a delay
// proportional to the reply length before the caller actually sends it —
// gated by msg.Meta["humanPacing"] (set only by
// assistant_executor.SendAssistantText, never on FlowBuilder text nodes in
// general) so this never changes behavior outside the Assistants plugin.
// Best-effort throughout: a PresenceEngine that errors or isn't implemented
// never blocks or fails the actual send that follows.
func (a *WhatsAppAdapter) applyHumanPacing(ctx context.Context, engine domain.WhatsAppEngine, whatsapp models.Whatsapp, msg OutboundMessage) {
	if msg.Meta["humanPacing"] != true || msg.Body == "" {
		return
	}
	if presenceEngine, ok := engine.(domain.PresenceEngine); ok {
		if err := presenceEngine.SendPresence(ctx, whatsapp, msg.To, "composing"); err != nil {
			log.Printf("[WhatsAppAdapter] presence composing (best-effort) failed: %v", err)
		}
	}
	time.Sleep(humanTypingDelay(len(msg.Body)))
}

// publish entrega o comando pelo canal da conexão quando ele sabe enviar mensagem rica; sem canal resolvido (call sites sem deps)
// publica direto no engine, como antes. O comando precisa levar o tipo na routing key: o engine despacha por ela.
func (a *WhatsAppAdapter) publish(ctx context.Context, sender domain.RichSender, conn *models.ChannelConnection, msg OutboundMessage, commandType string, payload map[string]interface{}) error {
	if sender != nil {
		return sender.SendRich(ctx, conn, commandType, payload)
	}
	routingKey := wbot.Command(msg.TenantID.String(), msg.SessionID, commandType)
	return a.rabbit.PublishCommand(routingKey, map[string]interface{}{"type": commandType, "payload": payload})
}

// releaseLock best-effort releases the dedup lock so a retry after a failed
// send can actually re-send (mirrors the AMQP publish-failure handling above).
func (a *WhatsAppAdapter) releaseLock(envID string) {
	if a.redis == nil || envID == "" {
		return
	}
	if err := a.redis.DelLock("wbot:msg:" + envID); err != nil {
		log.Printf("[WhatsAppAdapter] dedup lock release after engine-send failure failed (env=%s): %v", envID, err)
	}
}

// metaString reads a string value from the channel-specific Meta map.
func metaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
