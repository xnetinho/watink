package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/flow"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// dedupTTL: 24h para mensagens de qualquer canal (D6 / ADR 0033).
	channelDedupTTL = 24 * time.Hour
)

// ChannelInboundService é o serviço unificado de entrada para mensagens de todos os canais (Tarefa 3.2 / ADR 0033).
type ChannelInboundService struct {
	db             *gorm.DB
	redis          domain.RedisService
	ticketRepo     domain.TicketRepository
	contactRepo    domain.ContactRepository
	messageRepo    domain.MessageRepository
	receiveMessage *usecases.ReceiveMessageUseCase
	broadcast      domain.Broadcaster
	flowSkeleton   *flow.Skeleton
}

func NewChannelInboundService(
	db *gorm.DB,
	redis domain.RedisService,
	ticketRepo domain.TicketRepository,
	contactRepo domain.ContactRepository,
	messageRepo domain.MessageRepository,
	receiveMessage *usecases.ReceiveMessageUseCase,
	broadcast domain.Broadcaster,
	flowSkeleton *flow.Skeleton,
) *ChannelInboundService {
	return &ChannelInboundService{
		db:             db,
		redis:          redis,
		ticketRepo:     ticketRepo,
		contactRepo:    contactRepo,
		messageRepo:    messageRepo,
		receiveMessage: receiveMessage,
		broadcast:      broadcast,
		flowSkeleton:   flowSkeleton,
	}
}

// Receive é o ponto único de entrada para eventos de mensagem de qualquer canal.
// Realiza deduplicação atômica no Redis antes de processar.
func (s *ChannelInboundService) Receive(ctx context.Context, msg domain.InboundMessage) (*usecases.ReceiveMessageResult, error) {
	if msg.ExternalMessageID == "" {
		return nil, fmt.Errorf("externalMessageId is required for inbound message")
	}

	// 1. Deduplicação no Redis. A conversa entra na chave porque o id do canal só é único dentro dela (Telegram).
	dedupKey := ""
	// Edição reusa o id da original: o dedup a descartaria. Ela é idempotente (só regrava o corpo), então dispensa a trava.
	if s.redis != nil && !msg.Edited {
		dedupKey = "channel:msg:" + domain.StorageMessageID(msg.Channel, msg.ChannelConnectionID, conversationKeyOf(msg), msg.ExternalMessageID)
		acquired, err := s.redis.SetLock(dedupKey, "1", channelDedupTTL)
		if err != nil {
			log.Printf("[ChannelInbound] redis dedup lock check error (msg %s): %v", msg.ExternalMessageID, err)
			dedupKey = ""
		} else if !acquired {
			// Mensagem já processada anteriormente: descarta silenciosamente (idempotente)
			log.Printf("[ChannelInbound] duplicate message ignored: %s", dedupKey)
			return nil, nil
		}
	}

	res, err := s.processInbound(ctx, msg)
	if err != nil && dedupKey != "" {
		// Processamento falhou: libera a marca para o canal poder reentregar a
		// mensagem; sem isso ela seria descartada como duplicata por 24h.
		_ = s.redis.DelLock(dedupKey)
	}
	return res, err
}

// processInbound executa o pipeline de entrada de uma mensagem já deduplicada.
func (s *ChannelInboundService) processInbound(ctx context.Context, msg domain.InboundMessage) (*usecases.ReceiveMessageResult, error) {
	// 2. Se for canal WhatsApp, preserva paridade total delegando ao usecase existente
	if msg.Channel == domain.ChannelKindWhatsApp {
		return s.receiveMessage.Execute(ctx, usecases.ReceiveMessageInput{
			ID:            msg.ExternalMessageID,
			From:          msg.Sender.ExternalID,
			Body:          msg.Body,
			Type:          msg.MediaType,
			FromMe:        false,
			Timestamp:     msg.Timestamp.Unix(),
			PushName:      msg.Sender.Name,
			GroupName:     msg.Sender.GroupJID,
			QuotedMsgID:   msg.QuotedMessageID,
			ProfilePicURL: msg.Sender.AvatarURL,
			IsGroup:       msg.Sender.IsGroup,
			MediaURL:      msg.MediaURL,
			SessionID:     msg.ChannelConnectionID,
			TenantID:      msg.TenantID,
		})
	}

	// 3. Canais agnósticos (Telegram, Mercado Livre, OLX, Instagram, TikTok)
	// 3.1 Resolve ou cria o Contact através de ContactIdentity
	contact, _, err := s.contactRepo.FindOrCreateByIdentity(
		ctx,
		msg.TenantID,
		msg.Channel,
		msg.Sender.ExternalID,
		msg.Sender.Name,
		msg.Sender.AvatarURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve contact identity: %w", err)
	}

	// 3.2 Resolve ou cria Ticket pela conversationKey
	convKey := conversationKeyOf(msg)

	ticket, err := s.ticketRepo.FindOpenByConversation(ctx, msg.TenantID, msg.ChannelConnectionID, convKey)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup open ticket by conversation: %w", err)
	}

	connID := msg.ChannelConnectionID
	if ticket == nil {
		ticket, err = s.ticketRepo.FindOrCreatePending(ctx, &domain.Ticket{
			TenantID:            msg.TenantID,
			ContactID:           contact.ID,
			ChannelConnectionID: &connID,
			ConversationKey:     convKey,
			Status:              "pending",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create pending ticket: %w", err)
		}
	}

	// 3.3 Persiste a mensagem no Ticket
	mediaType := msg.MediaType
	if mediaType == "" {
		mediaType = "chat"
	}

	dataJSON := "{}"
	if msg.Forwarded || len(msg.Rich) > 0 {
		d := map[string]interface{}{}
		if msg.Forwarded {
			d["forwarded"] = true
		}
		if len(msg.Rich) > 0 {
			d["rich"] = msg.Rich
		}
		if b, err := json.Marshal(d); err == nil {
			dataJSON = string(b)
		}
	}

	messageID := domain.StorageMessageID(msg.Channel, msg.ChannelConnectionID, convKey, msg.ExternalMessageID)
	if msg.Edited {
		// Edição (inclusive localização ao vivo, que reenvia a posição a cada ~30s): atualiza a mensagem original e avisa a
		// tela, em vez de duplicar ou descartar. Se a original não existe, segue como mensagem nova.
		if orig, err := s.messageRepo.FindByID(ctx, messageID, msg.TenantID); err == nil && orig != nil {
			upd := map[string]interface{}{"body": msg.Body, "updatedAt": time.Now()}
			if err := s.messageRepo.Update(ctx, orig, upd); err != nil {
				return nil, fmt.Errorf("failed to update edited message: %w", err)
			}
			orig.Body = msg.Body
			if s.broadcast != nil {
				payload := map[string]interface{}{"action": "update", "message": orig}
				s.broadcast.EmitToRoom("/", "chat:"+strconv.Itoa(orig.TicketID), "appMessage", payload)
				s.broadcast.EmitToTenantRoom(msg.TenantID.String(), "appMessage", payload)
			}
			return &usecases.ReceiveMessageResult{Message: orig, Duplicate: true}, nil
		}
	}

	message := &domain.Message{
		ID:         messageID,
		DataJson:   dataJSON,
		ExternalID: msg.ExternalMessageID,
		Body:       msg.Body,
		Ack:        1, // recebida
		Read:       false,
		MediaType:  mediaType,
		MediaUrl:   msg.MediaURL,
		TicketID:   ticket.ID,
		ContactID:  &contact.ID,
		FromMe:     false,
		TenantID:   msg.TenantID,
		CreatedAt:  msg.Timestamp,
		UpdatedAt:  msg.Timestamp,
	}

	if rep, ok := s.messageRepo.(domain.MessageInsertReporter); ok {
		inserted, err := rep.CreateIfNotExistsReport(ctx, message)
		if err != nil {
			return nil, fmt.Errorf("failed to persist inbound message: %w", err)
		}
		if !inserted {
			return &usecases.ReceiveMessageResult{Contact: contact, Ticket: ticket, Message: message, Duplicate: true}, nil
		}
	} else if err := s.messageRepo.CreateIfNotExists(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to persist inbound message: %w", err)
	}

	// Atualiza o ticket com a última mensagem e timestamp
	lastMessage, lastKind := domain.LastMessagePreview(mediaType, msg.Body, "")
	_ = s.ticketRepo.Update(ctx, ticket, map[string]interface{}{
		"lastMessage":     lastMessage,
		"lastMessageType": lastKind,
		"updatedAt":       time.Now(),
	})

	res := &usecases.ReceiveMessageResult{
		Contact: contact,
		Ticket:  ticket,
		Message: message,
	}

	// 3.4 Broadcasts em tempo real (SSE)
	if s.broadcast != nil {
		room := "chat:" + strconv.Itoa(ticket.ID)
		ticketOut := withConnectionIdentity(ctx, s.db, ticket)
		msgPayload := map[string]interface{}{"action": "create", "message": message, "ticket": ticketOut, "contact": contact}
		s.broadcast.EmitToRoom("/", room, "appMessage", msgPayload)
		s.broadcast.EmitToTenantRoom(msg.TenantID.String(), "appMessage", msgPayload)
		s.broadcast.EmitToTenantRoom(msg.TenantID.String(), "ticket", map[string]interface{}{"action": "update", "ticket": ticketOut, "contact": contact})
	}

	// 3.5 Integração com FlowBuilder
	if s.flowSkeleton != nil {
		s.flowSkeleton.RouteInboundTicket(ctx, flow.InboundContext{
			TenantID:  msg.TenantID,
			Body:      msg.Body,
			FromMe:    false,
			EnvID:     msg.ExternalMessageID,
			Ticket:    ticket,
			Contact:   contact,
			MessageID: message.ID,
			MediaType: mediaType,
		})
	}

	return res, nil
}

// conversationKeyOf é a chave da conversa: a do canal ou, na falta dela, o remetente.
func conversationKeyOf(msg domain.InboundMessage) string {
	if msg.ConversationKey != "" {
		return msg.ConversationKey
	}
	return msg.Sender.ExternalID
}

// MessagePayloadToInboundMessage converte o MessagePayload legado do WhatsApp
// no contrato universal domain.InboundMessage (Tarefa 3.2 / ADR 0033).
func MessagePayloadToInboundMessage(p MessagePayload, sessionID int, tenantID uuid.UUID) domain.InboundMessage {
	senderJID := p.From
	if senderJID == "" {
		senderJID = p.Participant
	}
	ts := time.Now()
	if p.Timestamp > 0 {
		ts = time.Unix(p.Timestamp, 0)
	}
	return domain.InboundMessage{
		TenantID:            tenantID,
		ChannelConnectionID: sessionID,
		Channel:             domain.ChannelKindWhatsApp,
		ExternalMessageID:   p.ID,
		ConversationKey:     p.From,
		Sender: domain.InboundSender{
			ExternalID: senderJID,
			Name:       p.PushName,
			AvatarURL:  p.ProfilePicUrl,
			IsGroup:    p.IsGroup,
			GroupJID:   p.GroupName,
		},
		Body:            p.Body,
		MediaType:       p.Type,
		MediaURL:        p.MediaUrl,
		QuotedMessageID: p.QuotedMsgId,
		Timestamp:       ts,
	}
}
