package telegram

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"context"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Kind() domain.ChannelKind {
	return domain.ChannelKindTelegram
}

func (a *Adapter) Capabilities() domain.Capabilities {
	return domain.Capabilities{
		InitiateConversation: false, // Telegram bots não podem iniciar conversa sem comando/mensagem prévia do usuário
		ReplyWindow:          nil,   // Sem limite de janela temporal
		MaxTextBytes:         4096,  // Limite da Bot API do Telegram
		Media: []domain.MediaKind{
			domain.MediaKindImage,
			domain.MediaKindVideo,
			domain.MediaKindAudio,
			domain.MediaKindDocument,
		},
		Groups:       true,
		Typing:       true,
		ReadReceipts: false,
		Reactions:    true,
		Auth:         domain.AuthKindToken,
		HasPhone:     false,
	}
}

func (a *Adapter) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	token := input.Token
	if token == "" && conn.CredentialsEnc != "" {
		// Se já tiver credencial salva (reconexão)
		token = conn.CredentialsEnc
	}
	if token == "" {
		return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, fmt.Errorf("telegram bot token is required")
	}

	client := NewClient(token)
	user, err := client.GetMe(ctx)
	if err != nil {
		return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, fmt.Errorf("invalid telegram bot token: %w", err)
	}

	// Gera segredo de webhook se não houver
	secret := conn.WebhookSecret
	if secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = hex.EncodeToString(b)
	}

	// Monta URL pública de webhook (assumindo que o servidor expõe /webhooks/channels/telegram/:id)
	// Como o adapter roda no business, o webhook URL pode vir configurado no env ou derivado.
	// Por ora, registramos o secret e salvamos o token.
	conn.CredentialsEnc = token
	conn.WebhookSecret = secret
	conn.Status = "CONNECTED"
	conn.Config = fmt.Sprintf(`{"bot_username": "%s", "bot_id": %d, "first_name": "%s"}`, user.Username, user.ID, user.FirstName)

	return domain.ConnectResult{
		Status:    "CONNECTED",
		Connected: true,
	}, nil
}

func (a *Adapter) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	if conn.CredentialsEnc != "" {
		client := NewClient(conn.CredentialsEnc)
		_ = client.DeleteWebhook(ctx)
	}
	conn.Status = "DISCONNECTED"
	return nil
}

func (a *Adapter) Send(ctx context.Context, conn *models.ChannelConnection, msg OutboundMessageBridge) (domain.SendResult, error) {
	if conn.CredentialsEnc == "" {
		return domain.SendResult{}, fmt.Errorf("telegram connection has no token")
	}

	client := NewClient(conn.CredentialsEnc)
	var replyID int
	if msg.QuotedMessageID != "" {
		if idInt, err := strconv.Atoi(msg.QuotedMessageID); err == nil {
			replyID = idInt
		}
	}

	msgID, err := client.SendMessage(ctx, msg.ExternalID, msg.Body, replyID)
	if err != nil {
		return domain.SendResult{}, err
	}

	return domain.SendResult{
		ExternalMessageID: strconv.FormatInt(msgID, 10),
		Timestamp:         time.Now(),
		Status:            "sent",
	}, nil
}

// OutboundMessageBridge é um alias para evitar conflito de import se necessário
type OutboundMessageBridge = domain.OutboundMessage

// Structs para parse do webhook do Telegram
type TelegramUpdate struct {
	UpdateID      int64             `json:"update_id"`
	Message       *TelegramMessage  `json:"message,omitempty"`
	EditedMessage *TelegramMessage  `json:"edited_message,omitempty"`
}

type TelegramMessage struct {
	MessageID int64           `json:"message_id"`
	From      *TelegramUser   `json:"from,omitempty"`
	Chat      TelegramChat    `json:"chat"`
	Date      int64           `json:"date"`
	Text      string          `json:"text,omitempty"`
	Caption   string          `json:"caption,omitempty"`
	ReplyTo   *TelegramMessage `json:"reply_to_message,omitempty"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type TelegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "private", "group", "supergroup", "channel"
	Title string `json:"title,omitempty"`
}

func (a *Adapter) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	var update TelegramUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return nil, fmt.Errorf("failed to parse telegram update: %w", err)
	}

	msg := update.Message
	if msg == nil {
		msg = update.EditedMessage
	}
	if msg == nil {
		// Eventos não tratados (ex.: my_chat_member) retornam vazio sem erro
		return nil, nil
	}

	chatExternalID := strconv.FormatInt(msg.Chat.ID, 10)
	senderExternalID := chatExternalID
	senderName := "Telegram User"
	if msg.From != nil {
		senderExternalID = strconv.FormatInt(msg.From.ID, 10)
		senderName = msg.From.FirstName
		if msg.From.LastName != "" {
			senderName += " " + msg.From.LastName
		}
	}
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		if msg.Chat.Title != "" {
			senderName = msg.Chat.Title + " (" + senderName + ")"
		}
	}

	body := msg.Text
	if body == "" {
		body = msg.Caption
	}

	var quotedID string
	if msg.ReplyTo != nil {
		quotedID = strconv.FormatInt(msg.ReplyTo.MessageID, 10)
	}

	inbound := domain.InboundMessage{
		TenantID:            conn.TenantID,
		ChannelConnectionID: conn.ID,
		Channel:             domain.ChannelKindTelegram,
		ExternalMessageID:   strconv.FormatInt(msg.MessageID, 10),
		ConversationKey:     chatExternalID,
		Sender: domain.InboundSender{
			ExternalID: senderExternalID,
			Name:       senderName,
			IsGroup:    msg.Chat.Type == "group" || msg.Chat.Type == "supergroup",
			GroupJID:   chatExternalID,
		},
		Body:            body,
		MediaType:       "chat",
		QuotedMessageID: quotedID,
		Timestamp:       time.Unix(msg.Date, 0),
		RawPayload:      raw,
	}

	return []domain.InboundMessage{inbound}, nil
}
