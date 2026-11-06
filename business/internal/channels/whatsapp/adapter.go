package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

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
	callMediaBase  string
}

// CallMediaBase é o endereço do WebSocket interno de mídia das chamadas no engine (ex.: "ws://engine:8085").
type CallMediaBase string

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
		case CallMediaBase:
			a.callMediaBase = string(v)
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
		Forward:      true,
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

	msgID := msg.StorageID
	if msgID == "" {
		msgID = msg.ExternalID
	}
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
			payload["quotedJid"] = targetJID
			if qJID, ok := msg.Metadata["quotedJid"]; ok && qJID != "" {
				payload["quotedJid"] = qJID
			}
			if qBody, ok := msg.Metadata["quotedBody"]; ok && qBody != "" {
				payload["quotedBody"] = qBody
			}
			if q := msg.Quoted; q != nil {
				payload["quotedBody"] = q.Body
				if q.Media != nil {
					payload["quotedMedia"] = q.Media
				}
				if q.Participant != "" {
					payload["quotedJid"] = q.Participant
				}
			}
		}

		if msg.Forward != nil && !msg.Forward.AsNew {
			payload["forwarded"] = true
			payload["forwardScore"] = msg.Forward.Score + 1
		}

		if mentionsStr, ok := msg.Metadata["mentions"]; ok && mentionsStr != "" {
			parts := strings.Split(mentionsStr, ",")
			mentions := make([]string, 0, len(parts))
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					if !strings.Contains(trimmed, "@") {
						trimmed = trimmed + "@s.whatsapp.net"
					}
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
			fileName := msg.FileName
			if fileName == "" {
				fileName = msg.Metadata["fileName"]
			}
			if fileName != "" {
				payload["fileName"] = fileName
			}
			if msg.ViewOnce {
				payload["viewOnce"] = true
			}
			if msg.Voice {
				// O engine converte para OGG/Opus e envia como mensagem de voz.
				payload["mediaType"] = "voice"
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

// command publica um comando do engine para esta conexão. Falha se o adaptador não tem publisher.
func (a *Adapter) command(conn *models.ChannelConnection, commandType string, payload map[string]interface{}) error {
	if a.publisher == nil {
		return fmt.Errorf("whatsapp: command publisher not configured")
	}
	payload["sessionId"] = conn.ID
	routingKey := wbot.Command(conn.TenantID.String(), strconv.Itoa(conn.ID), commandType)
	if err := a.publisher.PublishCommand(routingKey, map[string]interface{}{"type": commandType, "payload": payload}); err != nil {
		return fmt.Errorf("whatsapp %s publish error: %w", commandType, err)
	}
	return nil
}

func newMessageID() string { return "3EB0" + fmt.Sprintf("%016X", time.Now().UnixNano()) }

var (
	_ domain.Reactor          = (*Adapter)(nil)
	_ domain.Editor           = (*Adapter)(nil)
	_ domain.PollVoter        = (*Adapter)(nil)
	_ domain.PresenceWatcher  = (*Adapter)(nil)
	_ domain.MediaDownloader  = (*Adapter)(nil)
	_ domain.HistoryRecoverer = (*Adapter)(nil)
	_ domain.ContactSyncer    = (*Adapter)(nil)
)

func (a *Adapter) React(_ context.Context, conn *models.ChannelConnection, to, targetMsgID string, targetFromMe bool, reaction string) error {
	return a.command(conn, wbot.CommandMessageReact, map[string]interface{}{
		"messageId": newMessageID(), "to": to, "targetMsgId": targetMsgID, "targetFromMe": targetFromMe, "reaction": reaction,
	})
}

func (a *Adapter) Edit(_ context.Context, conn *models.ChannelConnection, to, targetMsgID, body, mediaType string) error {
	return a.command(conn, wbot.CommandMessageEdit, map[string]interface{}{
		"messageId": newMessageID(), "to": to, "targetMsgId": targetMsgID, "body": body, "mediaType": mediaType,
	})
}

func (a *Adapter) VotePoll(_ context.Context, conn *models.ChannelConnection, chat, pollID, pollSender string, pollFromMe bool, options []string) error {
	return a.command(conn, wbot.CommandMessagePollVote, map[string]interface{}{
		"chat": chat, "pollId": pollID, "pollSender": pollSender, "pollFromMe": pollFromMe, "options": options,
	})
}

func (a *Adapter) WatchPresence(_ context.Context, conn *models.ChannelConnection, to string) error {
	return a.command(conn, wbot.CommandPresenceSubscribe, map[string]interface{}{"to": to})
}

func (a *Adapter) DownloadMedia(_ context.Context, conn *models.ChannelConnection, messageID, mediaType, proto string) error {
	return a.command(conn, wbot.CommandMediaDownload, map[string]interface{}{
		"messageId": messageID, "mediaType": mediaType, "mediaProto": proto,
	})
}

func (a *Adapter) RecoverHistory(_ context.Context, conn *models.ChannelConnection, params map[string]interface{}) error {
	return a.command(conn, wbot.CommandHistoryRecover, params)
}

func (a *Adapter) ImportContacts(_ context.Context, conn *models.ChannelConnection) error {
	return a.command(conn, wbot.CommandContactImport, map[string]interface{}{})
}

func (a *Adapter) SyncContact(_ context.Context, conn *models.ChannelConnection, number string) error {
	return a.command(conn, wbot.CommandContactSync, map[string]interface{}{"number": number})
}

func (a *Adapter) SaveContact(_ context.Context, conn *models.ChannelConnection, number, name string) error {
	return a.command(conn, wbot.CommandContactSave, map[string]interface{}{"number": number, "name": name})
}

// SendRich publica o comando rico como o chamador o montou. O sessionId é sempre o desta conexão (o engine o decodifica como
// inteiro), nunca o que veio no payload.
func (a *Adapter) SendRich(_ context.Context, conn *models.ChannelConnection, command string, payload map[string]interface{}) error {
	return a.command(conn, command, payload)
}

func (a *Adapter) CallCommand(_ context.Context, conn *models.ChannelConnection, command string, payload map[string]interface{}) error {
	return a.command(conn, command, payload)
}

// ErrCallMediaNotConfigured: o business não sabe onde está o canal de mídia das chamadas do engine.
var ErrCallMediaNotConfigured = errors.New("canal de áudio das chamadas não configurado (defina ENGINE_HOST no business)")

// OpenCallMedia abre o WebSocket de mídia da chamada no engine. Se CALLS_API_TOKEN ou INTERNAL_TOKEN estiver configurado, envia o
// cabeçalho X-Internal-Token para autenticação mútua interna.
func (a *Adapter) OpenCallMedia(ctx context.Context, _ *models.ChannelConnection, callID string) (domain.CallMediaStream, error) {
	if a.callMediaBase == "" {
		return nil, ErrCallMediaNotConfigured
	}
	opts := &websocket.DialOptions{}
	token := os.Getenv("CALLS_API_TOKEN")
	if token == "" {
		token = os.Getenv("INTERNAL_TOKEN")
	}
	if token != "" {
		opts.HTTPHeader = http.Header{"X-Internal-Token": []string{token}}
	}
	c, _, err := websocket.Dial(ctx, a.callMediaBase+"/calls/"+callID+"/audio", opts)
	if err != nil {
		return nil, err
	}
	return &wsStream{c: c}, nil
}

type wsStream struct{ c *websocket.Conn }

func (w *wsStream) Read(ctx context.Context) (bool, []byte, error) {
	typ, data, err := w.c.Read(ctx)
	return typ == websocket.MessageBinary, data, err
}

func (w *wsStream) Write(ctx context.Context, binary bool, data []byte) error {
	typ := websocket.MessageText
	if binary {
		typ = websocket.MessageBinary
	}
	return w.c.Write(ctx, typ, data)
}

func (w *wsStream) SetReadLimit(n int64) { w.c.SetReadLimit(n) }
func (w *wsStream) Close() error         { return w.c.CloseNow() }
