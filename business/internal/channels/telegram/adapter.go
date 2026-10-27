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
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
)

type Adapter struct {
	// inbound entrega ao serviço de entrada as mensagens recebidas por polling
	// (webhook entrega pelo controller). nil = polling não entrega.
	inbound InboundHandler
}

// InboundHandler entrega uma mensagem neutra ao pipeline de entrada.
type InboundHandler func(ctx context.Context, msg domain.InboundMessage) error

func NewAdapter(inbound InboundHandler) *Adapter {
	return &Adapter{inbound: inbound}
}

func (a *Adapter) Kind() domain.ChannelKind {
	return domain.ChannelKindTelegram
}

func (a *Adapter) Capabilities() domain.Capabilities {
	return domain.Capabilities{
		InitiateConversation: false,
		ReplyWindow:          nil,
		MaxTextBytes:         4096,
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
	if token == "" {
		stored, err := domain.ChannelCredentials(conn)
		if err == nil {
			token = stored
		}
	}
	if token == "" {
		return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, fmt.Errorf("telegram bot token is required")
	}

	client := NewClient(token)
	user, err := client.GetMe(ctx)
	if err != nil {
		return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, fmt.Errorf("invalid telegram bot token: %w", err)
	}

	secret := conn.WebhookSecret
	if secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = hex.EncodeToString(b)
	}

	mode := "webhook"
	if input.Params != nil && input.Params["mode"] != "" {
		mode = input.Params["mode"]
	} else if conn.Config != "" {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(conn.Config), &cfg) == nil {
			if m, ok := cfg["mode"].(string); ok && m != "" {
				mode = m
			}
		}
	}

	if mode == "polling" {
		_ = client.DeleteWebhook(ctx)
		globalPollingManager.Start(conn, a.inbound)
		conn.Config = fmt.Sprintf(`{"bot_username": "%s", "bot_id": %d, "first_name": "%s", "mode": "polling"}`, user.Username, user.ID, user.FirstName)
	} else {
		globalPollingManager.Stop(conn.ID)
		baseURL := "https://homolog.watink.com"
		if input.Params != nil && input.Params["baseUrl"] != "" {
			baseURL = input.Params["baseUrl"]
		}
		webhookURL := fmt.Sprintf("%s/api/v1/webhooks/channels/telegram/%d", baseURL, conn.ID)
		if err := client.SetWebhook(ctx, webhookURL, secret); err != nil {
			return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, fmt.Errorf("failed to set telegram webhook: %w", err)
		}
		conn.Config = fmt.Sprintf(`{"bot_username": "%s", "bot_id": %d, "first_name": "%s", "mode": "webhook", "webhook_url": "%s"}`, user.Username, user.ID, user.FirstName, webhookURL)
	}

	conn.WebhookSecret = secret
	conn.Status = "CONNECTED"

	return domain.ConnectResult{
		Status:    "CONNECTED",
		Connected: true,
	}, nil
}

func (a *Adapter) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	globalPollingManager.Stop(conn.ID)
	if token, err := domain.ChannelCredentials(conn); err == nil {
		_ = NewClient(token).DeleteWebhook(ctx)
	}
	conn.Status = "DISCONNECTED"
	return nil
}

func (a *Adapter) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	token, err := domain.ChannelCredentials(conn)
	if err != nil {
		return domain.SendResult{}, fmt.Errorf("telegram connection has no token")
	}

	// O destino é o chat da conversa: o contato do Telegram nasce por identidade e não tem number.
	chatID := msg.ConversationKey
	if chatID == "" {
		chatID = msg.ExternalID
	}
	if chatID == "" {
		return domain.SendResult{}, fmt.Errorf("telegram: conversa sem chat_id")
	}

	client := NewClient(token)
	var replyID int
	if msg.QuotedMessageID != "" {
		if idInt, err := strconv.Atoi(msg.QuotedMessageID); err == nil {
			replyID = idInt
		}
	}

	var msgID int64
	if msg.MediaURL != "" {
		mt := msg.MediaType
		if mt == "image" || mt == "photo" {
			msgID, err = client.SendPhoto(ctx, chatID, msg.MediaURL, msg.Body, replyID)
		} else {
			msgID, err = client.SendDocument(ctx, chatID, msg.MediaURL, msg.Body, replyID)
		}
	} else {
		body := msg.Body
		if len([]byte(body)) > 4096 {
			body = string([]byte(body)[:4096])
		}
		msgID, err = client.SendMessage(ctx, chatID, body, replyID)
	}

	if err != nil {
		conn.LastError = err.Error()
		return domain.SendResult{}, err
	}
	conn.LastError = ""

	return domain.SendResult{
		ExternalMessageID: strconv.FormatInt(msgID, 10),
		Timestamp:         time.Now(),
		Status:            "sent",
	}, nil
}

func (a *Adapter) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	if contact == nil {
		return "", fmt.Errorf("contact is nil")
	}
	return contact.Number, nil
}

type OutboundMessageBridge = domain.OutboundMessage

type TelegramUpdate struct {
	UpdateID      int64            `json:"update_id"`
	Message       *TelegramMessage `json:"message,omitempty"`
	EditedMessage *TelegramMessage `json:"edited_message,omitempty"`
}

type TelegramFileObject struct {
	FileID       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
}

type TelegramPhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size,omitempty"`
}

type TelegramMessage struct {
	MessageID int64               `json:"message_id"`
	From      *TelegramUser       `json:"from,omitempty"`
	Chat      TelegramChat        `json:"chat"`
	Date      int64               `json:"date"`
	Text      string              `json:"text,omitempty"`
	Caption   string              `json:"caption,omitempty"`
	ReplyTo   *TelegramMessage    `json:"reply_to_message,omitempty"`
	Photo     []TelegramPhotoSize `json:"photo,omitempty"`
	Document  *TelegramFileObject `json:"document,omitempty"`
	Video     *TelegramFileObject `json:"video,omitempty"`
	Audio     *TelegramFileObject `json:"audio,omitempty"`
	Voice     *TelegramFileObject `json:"voice,omitempty"`
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

	mediaType := "chat"
	var fileID string
	var mimeType string

	if len(msg.Photo) > 0 {
		mediaType = "image"
		// Pega a foto de maior resolução (última do array)
		fileID = msg.Photo[len(msg.Photo)-1].FileID
		mimeType = "image/jpeg"
	} else if msg.Document != nil {
		mediaType = "document"
		fileID = msg.Document.FileID
		mimeType = msg.Document.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	} else if msg.Video != nil {
		mediaType = "video"
		fileID = msg.Video.FileID
		mimeType = msg.Video.MimeType
		if mimeType == "" {
			mimeType = "video/mp4"
		}
	} else if msg.Audio != nil {
		mediaType = "audio"
		fileID = msg.Audio.FileID
		mimeType = msg.Audio.MimeType
		if mimeType == "" {
			mimeType = "audio/mpeg"
		}
	} else if msg.Voice != nil {
		mediaType = "audio"
		fileID = msg.Voice.FileID
		mimeType = msg.Voice.MimeType
		if mimeType == "" {
			mimeType = "audio/ogg"
		}
	}

	var mediaURL string
	if fileID != "" && conn.CredentialsEnc != "" {
		if token, err := domain.ChannelCredentials(conn); err == nil {
			client := NewClient(token)
			if filePath, err := client.GetFile(ctx, fileID); err == nil && filePath != "" {
				if reader, err := client.DownloadFile(ctx, filePath); err == nil {
					if savedURL, err := mediastore.SaveMediaReader(reader, mimeType); err == nil {
						mediaURL = savedURL
					}
					_ = reader.Close()
				}
			}
		}
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
		MediaType:       mediaType,
		MediaURL:        mediaURL,
		QuotedMessageID: quotedID,
		Timestamp:       time.Unix(msg.Date, 0),
		RawPayload:      raw,
	}

	return []domain.InboundMessage{inbound}, nil
}
