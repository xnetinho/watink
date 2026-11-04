package services

import (
	"context"
	"encoding/json"
	"fmt"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"log"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/flow"
	"github.com/alltomatos/watinkdev/business/internal/mediawait"
	"github.com/alltomatos/watinkdev/business/internal/plugins"
	"github.com/alltomatos/watinkdev/business/pkg/safego"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	amqp "github.com/streadway/amqp"
	"gorm.io/gorm"
)

type EventListener struct {
	sessions       domain.ChannelSessionRepository
	messages       domain.MessageRepository
	contacts       domain.ContactRepository
	tickets        domain.TicketRepository
	receiveMessage *usecases.ReceiveMessageUseCase
	broadcast      domain.Broadcaster
	db             *gorm.DB
	flowSkeleton   *flow.Skeleton
	mediaWaiter    *mediawait.Waiter
	publisher      domain.CommandPublisher
}

// publisher is the minimal domain.CommandPublisher slice AssistantRuntime
// needs to trigger an on-demand media.download itself (áudio recebido por um
// Assistant com AcceptsAudio=true) — same contract as message_media.go's
// HTTP handler, just invoked programmatically instead of via request.
func NewEventListener(sessions domain.ChannelSessionRepository, messages domain.MessageRepository, contacts domain.ContactRepository, tickets domain.TicketRepository, rm *usecases.ReceiveMessageUseCase, broadcast domain.Broadcaster, db *gorm.DB, registry *flow.ChannelRegistry, redis domain.RedisService, publisher domain.CommandPublisher, mediaWaiter *mediawait.Waiter) *EventListener {
	// SetAssistantRuntime: sem isto, todo assistant-node de toda mensagem
	// real recebida (message.received) executa com st.AssistantRuntime==nil
	// e assistantExecutor.Execute degrada silenciosamente pra "sem
	// assistantId/runtime" (Advance, sem edge de saída, run "completed" sem
	// nunca chamar o Mode do Assistant) -- o plugin "Assistentes de IA"
	// nunca respondia UMA mensagem real sequer, apesar de rodar via
	// /flows/:id/run (routes.go monta um Skeleton SEPARADO, com o runtime
	// wireado, só pro endpoint manual). Bug reproduzido ao vivo em homolog:
	// FlowRun entrava no nó assistant e completava em ~0ms sem nenhuma
	// query em Assistants/AssistantRouterOptions, sem enviar nada.
	skeleton := flow.NewSkeleton(db, registry, redis)
	skeleton.SetAssistantRuntime(plugins.NewAssistantRuntime(db, publisher, mediaWaiter))
	return &EventListener{sessions: sessions, messages: messages, contacts: contacts, tickets: tickets, receiveMessage: rm, broadcast: domain.BroadcastOrNop(broadcast), db: db, flowSkeleton: skeleton, mediaWaiter: mediaWaiter, publisher: publisher}
}

// ConfigureKnowledge wires the flow skeleton's RAG retriever/responder (see
// flow.Skeleton.SetRetriever/SetResponder) — called from main.go right after
// construction so the native pgvector RAG applies to the real inbound
// WhatsApp message path, not just the on-demand endpoints wired in routes.go.
func (el *EventListener) ConfigureKnowledge(retriever flow.Retriever, responder flow.AgentResponder) {
	el.flowSkeleton.SetRetriever(retriever)
	el.flowSkeleton.SetResponder(responder)
}

// bcast returns a nil-safe broadcaster — tests that construct EventListener
// directly with broadcast=nil still get a no-op instead of a panic.
func (el *EventListener) bcast() domain.Broadcaster {
	return domain.BroadcastOrNop(el.broadcast)
}

// eventRoutingKeys são os eventos do engine que o business consome. Todo evento do contrato que o engine publica e que
// tem tratador precisa estar aqui: o que falta é descartado em silêncio pelo broker (foi o que deixou o contato em
// "Gravando na agenda do celular..." para sempre, com o evento contact.saved nunca entregue).
func eventRoutingKeys() []string {
	return []string{
		wbot.EventPattern(wbot.EventEngineReady),
		wbot.EventPattern(wbot.EventSessionQRCode),
		wbot.EventPattern(wbot.EventSessionPairingCode),
		wbot.EventPattern(wbot.EventSessionStatus),
		wbot.EventPattern(wbot.EventSessionHistorySync),
		wbot.EventPattern(wbot.EventSessionRisk),
		wbot.EventPattern(wbot.EventMessageReceived),
		wbot.EventPattern(wbot.EventMessageAck),
		wbot.EventPattern(wbot.EventMessageRevoke),
		wbot.EventPattern(wbot.EventMessageEdited),
		wbot.EventPattern(wbot.EventMessageReaction),
		wbot.EventPattern(wbot.EventMessageMedia),
		wbot.EventPattern(wbot.EventContactUpdate),
		wbot.EventPattern(wbot.EventContactImport),
		wbot.EventPattern(wbot.EventContactSaved),
		wbot.EventPattern(wbot.EventSessionJIDRegistered),
		wbot.EventPattern(wbot.EventMessagePollVote),
		wbot.EventPattern(wbot.EventContactPresence),
	}
}

func StartEventListener(rabbitMQ *RabbitMQService, eventListener *EventListener) {
	routingKeys := eventRoutingKeys()
	// Eventos call.* NÃO entram aqui: têm fila e consumidor próprios em
	// internal/calls (Service.Start), para não esperarem atrás das mensagens.

	err := rabbitMQ.ConsumeEvents("api.events.process.go", routingKeys, func(d amqp.Delivery) error {
		var env EventEnvelope
		if err := json.Unmarshal(d.Body, &env); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			return err
		}

		if env.Type == wbot.EventEngineReady {
			return eventListener.handleEngineReady(context.Background())
		}
		tid, err := uuid.Parse(env.TenantID)
		if err != nil {
			return fmt.Errorf("invalid tenantId %q: %w", env.TenantID, err)
		}

		log.Printf("[EventListener] Event received: %s (Tenant: %s)", env.Type, env.TenantID)

		// Todo evento roda no escopo da empresa do envelope (RLS, ADR 0034): o banco só enxerga as linhas dela.
		ctx, done := tenantdb.TenantCtx(context.Background(), eventListener.db, tid)
		defer done()
		switch env.Type {
		case "session.qrcode":
			return eventListener.handleQrCode(ctx, env.Payload, tid)
		case "session.pairing_code":
			return eventListener.handlePairingCode(ctx, env.Payload, tid)
		case "session.status":
			return eventListener.handleSessionStatus(ctx, env.Payload, tid)
		case "session.history_sync":
			return eventListener.handleHistorySync(ctx, env.Payload, tid)
		case "session.risk":
			return eventListener.handleSessionRisk(ctx, env.Payload, tid)
		case "message.received":
			var p MessageReceivedPayload
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				return err
			}
			return eventListener.processMessage(ctx, p.Message, p.SessionID, tid)
		case "message.ack":
			return eventListener.handleMessageAck(ctx, env.Payload, tid)
		case "message.revoke":
			return eventListener.handleMessageRevoke(ctx, env.Payload, tid)
		case "message.edited":
			return eventListener.handleMessageEdited(ctx, env.Payload, tid)
		case "message.reaction":
			return eventListener.handleMessageReaction(ctx, env.Payload, tid)
		case "message.media":
			// O consumidor de eventos é um único loop sequencial (ConsumeEvents
			// processa uma Delivery por vez para TODOS os eventos do tenant —
			// mensagens, recibos, presença, mídia). Um tenant com tráfego alto
			// (muitos grupos ativos) enfileira o evento message.media atrás de
			// centenas de outros eventos, atrasando o Fulfill do mediawait em
			// minutos mesmo quando o download em si (engine-go) levou <1s —
			// medido ao vivo em homolog. handleMediaDownloaded só libera um
			// mediawait.Waiter (map com mutex, sem dependência de ordem com
			// outros eventos), então rodar em goroutine própria é seguro e
			// evita esse atraso sem mudar o contrato — ack imediato (best-effort,
			// mesmo padrão de media.download no engine-go).
			bgCtx, bgDone := tenantdb.Detach(ctx)
			safego.Go("event.handleMediaDownloaded", func() {
				defer bgDone()
				if err := eventListener.handleMediaDownloaded(bgCtx, env.Payload, tid); err != nil {
					log.Printf("[EventListener] handleMediaDownloaded falhou: %v", err)
				}
			})
			return nil
		case "contact.update":
			if err := handleContactUpdate(ctx, eventListener.contacts, eventListener.bcast(), env.Payload, tid); err != nil {
				return err
			}
			// Mesmo evento de enriquecimento — também atualiza a foto da PRÓPRIA
			// conexão quando o número bate (não é um contato, é o número da sessão).
			return syncConnectionProfilePic(ctx, eventListener.sessions, eventListener.bcast(), env.Payload, tid)
		case "contact.import":
			return handleContactImport(ctx, eventListener.contacts, env.Payload, tid)
		case "contact.saved":
			return handleContactSaved(ctx, eventListener.contacts, eventListener.bcast(), env.Payload, tid)
		case "session.jid_registered":
			return handleJIDRegistered(ctx, eventListener.sessions, env.Payload, tid)
		case "message.poll_vote":
			return eventListener.handlePollVote(ctx, env.Payload, tid)
		case "contact.presence":
			return eventListener.handleContactPresence(env.Payload, tid)
		default:
			return nil
		}
	})

	if err != nil {
		log.Printf("Error starting event listener: %v", err)
	}
}

// HandleInboundMessage is the exported entry point for non-AMQP transports
// (e.g. a webhook-based engine) to feed an inbound message through the
// exact same pipeline (ReceiveMessageUseCase + broadcasts + FlowBuilder
// routing) as the engine-go AMQP consumer.
func (el *EventListener) HandleInboundMessage(ctx context.Context, p MessagePayload, rawSessionID string, tenantID uuid.UUID) error {
	return el.processMessage(ctx, p, rawSessionID, tenantID)
}

// HandleSessionStatusEvent is the exported entry point for non-AMQP
// transports to apply a session status transition (connected/disconnected/
// banned/...) through the same logic as the engine-go AMQP consumer
// (handleSessionStatus), including auto-isolation of a banned proxy.
func (el *EventListener) HandleSessionStatusEvent(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	return el.handleSessionStatus(ctx, payload, tenantID)
}

// HandlePollVoteEvent is the exported entry point for non-AMQP transports to
// record a poll vote through the same logic as the engine-go AMQP consumer
// (handlePollVote) — persists a PollResult only when the originating
// QuickAnswer has capture_results:true; never touches FlowBuilder/ticket state
// (mirrors whatsmeow: a poll vote is metadata, not an inbound message).
func (el *EventListener) HandlePollVoteEvent(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	return el.handlePollVote(ctx, payload, tenantID)
}

// HandleMessageAckEvent é a porta exportada do recibo de uma mensagem (mesma lógica do consumidor).
func (el *EventListener) HandleMessageAckEvent(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	return el.handleMessageAck(ctx, payload, tenantID)
}

// HandleMediaDownloadedEvent é a porta exportada do resultado de um download de mídia do engine (mesma lógica do consumidor).
func (el *EventListener) HandleMediaDownloadedEvent(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	return el.handleMediaDownloaded(ctx, payload, tenantID)
}

// HandleSessionRiskEvent is the exported entry point for non-AMQP transports
// to apply a ban/throttle risk signal (401/403/429/463) through the same
// logic as the engine-go AMQP consumer (handleSessionRisk):
// persists lastRiskCode/Action/Message/At on the connection, auto-isolates
// the proxy for 429/463, and broadcasts whatsappSessionRisk to the tenant.
func (el *EventListener) HandleSessionRiskEvent(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	return el.handleSessionRisk(ctx, payload, tenantID)
}

func (el *EventListener) processMessage(ctx context.Context, p MessagePayload, rawSessionID string, tenantID uuid.UUID) error {
	sessionID := getSessionID(rawSessionID)

	result, err := el.receiveMessage.Execute(ctx, usecases.ReceiveMessageInput{
		ID:            p.ID,
		From:          p.From,
		Body:          p.Body,
		Type:          p.Type,
		FromMe:        p.FromMe,
		Timestamp:     p.Timestamp,
		PushName:      p.PushName,
		GroupName:     p.GroupName,
		QuotedMsgID:   p.QuotedMsgId,
		ProfilePicURL: p.ProfilePicUrl,
		SenderPicURL:  p.SenderPicUrl,
		IsLID:         p.IsLid,
		Participant:   p.Participant,
		ChatPN:        p.ChatPn,
		IsGroup:       p.IsGroup,
		IsCommunity:   p.IsCommunity,
		IsSubGroup:    p.IsSubGroup,
		MediaURL:      p.MediaUrl,
		MediaData:     p.MediaData,
		Mimetype:      p.Mimetype,
		Thumbnail:     p.Thumbnail,
		MediaProto:    p.MediaProto,
		Forwarded:     p.Forwarded,
		ForwardScore:  p.ForwardScore,
		Rich:          p.Rich,
		FileLength:    p.FileLength,
		SessionID:     sessionID,
		TenantID:      tenantID,
	})
	if err != nil {
		return err
	}
	if result.Duplicate {
		return nil // reentrega: já foi anunciada e roteada na primeira vez
	}

	room := "chat:" + strconv.Itoa(result.Ticket.ID)
	attachQuote(ctx, el.messages, result.Message)
	ticketOut := withConnectionIdentity(ctx, el.db, result.Ticket)
	msgPayload := map[string]interface{}{"action": "create", "message": result.Message, "ticket": ticketOut, "contact": result.Contact}
	el.bcast().EmitToRoom("/", room, "appMessage", msgPayload)
	el.bcast().EmitToTenantRoom(tenantID.String(), "appMessage", msgPayload)
	el.bcast().EmitToTenantRoom(tenantID.String(), "ticket", map[string]interface{}{"action": "update", "ticket": ticketOut, "contact": result.Contact})

	// Mídia pequena baixa sozinha pelo mesmo comando do botão; ao terminar, o handler de message.media troca o
	// "pendente" pela mídia real. Falha aqui não derruba o recebimento: a mensagem fica com o botão manual.
	if el.publisher != nil && shouldAutoDownload(p.Type, p.FileLength, autoDownloadMaxBytes(), p.MediaProto != "") {
		cmd := map[string]interface{}{
			"id": uuid.New().String(), "timestamp": time.Now().UnixMilli(), "tenantId": tenantID.String(), "type": "media.download",
			"payload": map[string]interface{}{"sessionId": sessionID, "messageId": result.Message.ID, "mediaType": p.Type, "mediaProto": p.MediaProto},
		}
		if err := el.publisher.PublishCommand(wbot.Command(tenantID.String(), strconv.Itoa(sessionID), wbot.CommandMediaDownload), cmd); err != nil {
			log.Printf("[EventListener] auto-download da mídia %s falhou ao enfileirar: %v", result.Message.ID, err)
		}
	}

	// FlowBuilder FASE 1 seam: route the inbound through the runtime AFTER the
	// real-time broadcasts, so its DB round-trip never delays SSE delivery. The
	// dispatcher does resume-first / opt-out / trigger→StartFlow, tenant-aware
	// (WHERE "tenantId" manual). EnvID = inbound message id (redelivery dedup).
	if el.flowSkeleton != nil {
		el.flowSkeleton.RouteInboundTicket(ctx, flow.InboundContext{
			TenantID:      tenantID,
			Body:          p.Body,
			FromMe:        p.FromMe,
			EnvID:         p.ID,
			Ticket:        result.Ticket,
			Contact:       result.Contact,
			MentionedJIDs: p.MentionedJids,
			MessageID:     result.Message.ID,
			MediaType:     p.Type,
			Mimetype:      p.Mimetype,
		})
	}

	return nil
}

// HandleEngineReady é o ponto de entrada exportado de handleEngineReady (restauração das sessões ativas), no mesmo
// padrão dos demais Handle* usados por transportes que não são o consumidor AMQP e pelos testes de integração.
func (el *EventListener) HandleEngineReady(ctx context.Context) error {
	return el.handleEngineReady(ctx)
}

// handleEngineReady asks the business-owned connection catalog to restore active sessions.
// The engine never queries public business tables; it only announces readiness and receives commands.
func (el *EventListener) handleEngineReady(ctx context.Context) error {
	if el.db == nil || el.publisher == nil {
		return nil
	}
	systemCtx, systemDone := tenantdb.SystemCtx(ctx, el.db)
	defer systemDone()
	var tenantIDs []uuid.UUID
	if err := el.db.WithContext(systemCtx).Table(`"Tenants"`).Pluck("id", &tenantIDs).Error; err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		tenantCtx, tenantDone := tenantdb.TenantCtx(ctx, el.db, tenantID)
		sessions, err := el.sessions.FindAll(tenantCtx, tenantID)
		if err != nil {
			tenantDone()
			return err
		}
		for _, session := range sessions {
			if session.Status != "CONNECTED" && session.Status != "OPENING" && session.Status != "QRCODE" {
				continue
			}
			model, err := el.sessions.FindByIDDetail(tenantCtx, session.ID, tenantID)
			if err != nil || model == nil {
				continue
			}
			payload := map[string]interface{}{
				"type": wbot.CommandSessionStart,
				"payload": map[string]interface{}{
					"sessionId": session.ID,
					"name":      session.Name,
					"wid":       model.Wid,
					// A política de presença precisa viajar também na restauração após o engine reiniciar.
					"presencePolicy":      model.PresencePolicy,
					"presenceIdleMinutes": model.PresenceIdleMinutes,
				},
			}
			if err := el.publisher.PublishCommand(wbot.Command(tenantID.String(), strconv.Itoa(session.ID), wbot.CommandSessionStart), payload); err != nil {
				tenantDone()
				return err
			}
		}
		tenantDone()
	}
	return nil
}
