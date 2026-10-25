package services

import (
	"context"
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
	if s.redis != nil {
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

	message := &domain.Message{
		ID:         domain.StorageMessageID(msg.Channel, msg.ChannelConnectionID, convKey, msg.ExternalMessageID),
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

	if err := s.messageRepo.CreateIfNotExists(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to persist inbound message: %w", err)
	}

	// Atualiza o ticket com a última mensagem e timestamp
	_ = s.ticketRepo.Update(ctx, ticket, map[string]interface{}{
		"lastMessage": msg.Body,
		"updatedAt":   time.Now(),
	})

	res := &usecases.ReceiveMessageResult{
		Contact: contact,
		Ticket:  ticket,
		Message: message,
	}

	// 3.4 Broadcasts em tempo real (SSE)
	if s.broadcast != nil {
		room := "chat:" + strconv.Itoa(ticket.ID)
		msgPayload := map[string]interface{}{"action": "create", "message": message, "ticket": ticket, "contact": contact}
		s.broadcast.EmitToRoom("/", room, "appMessage", msgPayload)
		s.broadcast.EmitToTenantRoom(msg.TenantID.String(), "appMessage", msgPayload)
		s.broadcast.EmitToTenantRoom(msg.TenantID.String(), "ticket", map[string]interface{}{"action": "update", "ticket": ticket, "contact": contact})
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
