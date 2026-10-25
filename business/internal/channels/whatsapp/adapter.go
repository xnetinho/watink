package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
)

// SessionManager abstracts session lifecycle management (start, stop).
type SessionManager interface {
	StartWhatsAppSession(whatsapp models.Whatsapp, usePairingCode bool, phoneNumber string, force bool) error
	StopWhatsAppSession(whatsapp models.Whatsapp) error
}

// Adapter implementa domain.Channel para o WhatsApp (Tarefa 4.5 / ADR 0033).
// Suporta conectar, desconectar, envio de texto com citação e menções, envio de mídia com MIME,
// e consulta de status.
type Adapter struct {
	engineResolver domain.WhatsAppEngineResolver
	sessionManager SessionManager
	publisher      domain.CommandPublisher
}

// NewAdapter instancia o adaptador WhatsApp. Aceita opcionalmente um SessionManager e um CommandPublisher.
func NewAdapter(resolver domain.WhatsAppEngineResolver, options ...interface{}) *Adapter {
	a := &Adapter{
		engineResolver: resolver,
	}
	for _, opt := range options {
		switch v := opt.(type) {
		case SessionManager:
			a.sessionManager = v
		case domain.CommandPublisher:
			a.publisher = v
		}
	}
	if a.sessionManager == nil {
		if sm, ok := resolver.(SessionManager); ok {
			a.sessionManager = sm
		}
	}
	return a
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
		Voice:        true,
		Video:        true,
		Auth:         domain.AuthKindQR,
		HasPhone:     true,
	}
}

func (a *Adapter) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	if a.sessionManager != nil {
		wa := models.Whatsapp{
			ID:         conn.ID,
			TenantID:   conn.TenantID,
			Name:       conn.Name,
			Status:     conn.Status,
			EngineType: "whatsmeow",
		}
		usePairing := false
		phone := ""
		if input.Params != nil {
			usePairing = input.Params["usePairingCode"] == "true"
			phone = input.Params["phoneNumber"]
		}
		if err := a.sessionManager.StartWhatsAppSession(wa, usePairing, phone, false); err != nil {
			return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, err
		}
		return domain.ConnectResult{Status: "OPENING", Connected: true}, nil
	}
	return domain.ConnectResult{
		Status:    conn.Status,
		Connected: conn.Status == "CONNECTED",
	}, nil
}

func (a *Adapter) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	if a.sessionManager != nil {
		wa := models.Whatsapp{
			ID:         conn.ID,
			TenantID:   conn.TenantID,
			Name:       conn.Name,
			Status:     conn.Status,
			EngineType: "whatsmeow",
		}
		return a.sessionManager.StopWhatsAppSession(wa)
	}
	return nil
}

func (a *Adapter) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	targetJID := msg.ExternalID
	if targetJID == "" {
		targetJID = msg.ConversationKey
	}
	if targetJID == "" {
		return domain.SendResult{}, fmt.Errorf("whatsapp: empty destination (externalId/conversationKey)")
	}

	msgID := msg.ExternalID
	if msgID == "" || msgID == targetJID {
		msgID = "3EB0" + fmt.Sprintf("%016X", time.Now().UnixNano())
	}

	// 1. Envio direto via CommandPublisher (RabbitMQ) se disponível
	if a.publisher != nil {
		commandType := wbot.CommandMessageText
		payload := map[string]interface{}{
			"sessionId": conn.ID,
			"messageId": msgID,
			"to":        targetJID,
			"body":      msg.Body,
		}

		if msg.QuotedMessageID != "" {
			payload["quotedMsgId"] = msg.QuotedMessageID
			if qJID, ok := msg.Metadata["quotedJid"]; ok && qJID != "" {
				payload["quotedJid"] = qJID
			} else {
				payload["quotedJid"] = targetJID
			}
			if qBody, ok := msg.Metadata["quotedBody"]; ok && qBody != "" {
				payload["quotedBody"] = qBody
			}
		}

		if mentionsStr, ok := msg.Metadata["mentions"]; ok && mentionsStr != "" {
			parts := strings.Split(mentionsStr, ",")
			mentions := make([]string, 0, len(parts))
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					mentions = append(mentions, trimmed)
				}
			}
			if len(mentions) > 0 {
				payload["mentions"] = mentions
			}
		}

		if msg.MediaType != "" && msg.MediaType != "chat" {
			commandType = wbot.CommandMessageMedia
			payload["mediaType"] = msg.MediaType
			payload["mediaUrl"] = msg.MediaURL
			if mime, ok := msg.Metadata["mimeType"]; ok && mime != "" {
				payload["mimeType"] = mime
			}
			if fn, ok := msg.Metadata["fileName"]; ok && fn != "" {
				payload["fileName"] = fn
			}
			if len(msg.MediaData) > 0 {
				payload["mediaData"] = base64.StdEncoding.EncodeToString(msg.MediaData)
			} else if b64, ok := msg.Metadata["mediaData"]; ok && b64 != "" {
				payload["mediaData"] = b64
			}
		}

		command := map[string]interface{}{
			"type":    commandType,
			"payload": payload,
		}
		routingKey := wbot.Command(conn.TenantID.String(), strconv.Itoa(conn.ID), commandType)
		if err := a.publisher.PublishCommand(routingKey, command); err != nil {
			return domain.SendResult{}, fmt.Errorf("whatsapp send publish error: %w", err)
		}

		return domain.SendResult{
			ExternalMessageID: msgID,
			Timestamp:         time.Now(),
			Status:            "sent",
		}, nil
	}

	// 2. Fallback via engineResolver se CommandPublisher não foi injetado
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

	if msg.MediaType != "" && msg.MediaType != "chat" {
		mime := ""
		if m, ok := msg.Metadata["mimeType"]; ok {
			mime = m
		}
		effID, err := eng.SendMedia(ctx, wa, targetJID, msgID, msg.MediaType, msg.MediaURL, mime)
		if err != nil {
			return domain.SendResult{}, err
		}
		return domain.SendResult{
			ExternalMessageID: effID,
			Timestamp:         time.Now(),
			Status:            "sent",
		}, nil
	}

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
