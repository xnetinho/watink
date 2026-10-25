package whatsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

// Adapter implementa domain.Channel para o WhatsApp (Tarefa 4.1 / ADR 0033).
// Encapsula o WhatsAppEngine/enginego existente, provendo paridade estrita.
type Adapter struct {
	engineResolver domain.WhatsAppEngineResolver
}

func NewAdapter(resolver domain.WhatsAppEngineResolver) *Adapter {
	return &Adapter{
		engineResolver: resolver,
	}
}

func (a *Adapter) Kind() domain.ChannelKind {
	return domain.ChannelKindWhatsApp
}

func (a *Adapter) Capabilities() domain.Capabilities {
	return domain.Capabilities{
		InitiateConversation: true,
		ReplyWindow:          nil, // WhatsApp não restringe janela de atendimento
		MaxTextBytes:         65535,
		Media: []domain.MediaKind{
			domain.MediaKindImage,
			domain.MediaKindVideo,
			domain.MediaKindAudio,
			domain.MediaKindDocument,
			domain.MediaKindSticker,
		},
		Groups:       true,
		Typing:       true,
		ReadReceipts: true,
		Reactions:    true,
		Auth:         domain.AuthKindQR,
		HasPhone:     true,
	}
}

func (a *Adapter) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	// A gestão de ciclo de vida do WhatsApp (QR code, pairing) continua orquestrada via sessionService
	return domain.ConnectResult{
		Status: conn.Status,
	}, nil
}

func (a *Adapter) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	return nil
}

func (a *Adapter) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	if a.engineResolver == nil {
		return domain.SendResult{}, fmt.Errorf("whatsapp engine resolver not configured")
	}

	wa := models.Whatsapp{
		ID:         conn.ID,
		TenantID:   conn.TenantID,
		EngineType: "whatsmeow",
	}

	eng, err := a.engineResolver.EngineFor(wa)
	if err != nil {
		return domain.SendResult{}, fmt.Errorf("failed to resolve whatsapp engine: %w", err)
	}

	targetJID := msg.ExternalID
	if targetJID == "" {
		targetJID = msg.ConversationKey
	}

	msgID := "3EB0" + fmt.Sprintf("%016X", time.Now().UnixNano())

	// Se for envio de mídia
	if msg.MediaType != "" && msg.MediaType != "chat" {
		effID, err := eng.SendMedia(ctx, wa, targetJID, msgID, msg.MediaType, msg.MediaURL, "")
		if err != nil {
			return domain.SendResult{}, err
		}
		return domain.SendResult{
			ExternalMessageID: effID,
			Timestamp:         time.Now(),
			Status:            "sent",
		}, nil
	}

	// Envio de texto padrão
	effID, err := eng.SendText(ctx, wa, targetJID, msgID, msg.Body)
	if err != nil {
		return domain.SendResult{}, err
	}

	return domain.SendResult{
		ExternalMessageID: effID,
		Timestamp:         time.Now(),
		Status:            "sent",
	}, nil
}

func (a *Adapter) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	if contact == nil {
		return "", fmt.Errorf("contact is nil")
	}
	if contact.Lid != nil && *contact.Lid != "" {
		return *contact.Lid, nil
	}
	if contact.Number != "" {
		return contact.Number + "@s.whatsapp.net", nil
	}
	return "", fmt.Errorf("contact has neither number nor lid")
}

func (a *Adapter) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	// Mensagens de entrada do WhatsApp chegam via AMQP/EventListener e são convertidas via MessagePayloadToInboundMessage
	return nil, nil
}
