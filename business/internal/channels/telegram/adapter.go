package telegram

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
		Forward:      true,
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
	if f := msg.Forward; f != nil && !f.AsNew {
		// Encaminhamento de verdade: o Telegram só encaminha de um chat que o MESMO bot enxerga.
		if !f.SameConnection || f.ExternalMessageID == "" || f.FromConversationKey == "" {
			return domain.SendResult{}, domain.ErrForwardUnsupported
		}
		msgID, err = client.ForwardMessage(ctx, chatID, f.FromConversationKey, f.ExternalMessageID)
	} else if msg.MediaURL != "" {
		switch msg.MediaType {
		case "image", "photo":
			msgID, err = client.SendPhoto(ctx, chatID, msg.MediaURL, msg.Body, replyID)
		case "video":
			msgID, err = client.SendVideo(ctx, chatID, msg.MediaURL, msg.Body, replyID)
		case "audio":
			msgID, err = client.SendAudio(ctx, chatID, msg.MediaURL, msg.Body, replyID)
		case "sticker":
			msgID, err = client.SendSticker(ctx, chatID, msg.MediaURL, replyID)
		default:
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
	VideoNote *TelegramFileObject `json:"video_note,omitempty"`
	Animation *TelegramFileObject `json:"animation,omitempty"`
	Sticker   *struct {
		TelegramFileObject
		IsAnimated bool `json:"is_animated"`
		IsVideo    bool `json:"is_video"`
	} `json:"sticker,omitempty"`
	Location *struct {
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		LivePeriod int     `json:"live_period,omitempty"`
	} `json:"location,omitempty"`
	Venue *struct {
		Title   string `json:"title"`
		Address string `json:"address"`
	} `json:"venue,omitempty"`
	Contact *struct {
		PhoneNumber string `json:"phone_number"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
	} `json:"contact,omitempty"`
	ForwardOrigin json.RawMessage `json:"forward_origin,omitempty"`
	EditDate      int64           `json:"edit_date,omitempty"`
	Poll          *struct {
		Question       string `json:"question"`
		AllowsMultiple bool   `json:"allows_multiple_answers"`
		IsAnonymous    bool   `json:"is_anonymous"`
		Type           string `json:"type"`
		Options        []struct {
			Text string `json:"text"`
		} `json:"options"`
	} `json:"poll,omitempty"`
}

// telegramKnownFields são os campos de uma mensagem que o adapter trata ou que não são conteúdo (metadados). Qualquer
// outro campo preenchido (dice, game, invoice, novos tipos que o Telegram criar...) vira um aviso de "não suportado".
var telegramKnownFields = map[string]bool{
	"message_id": true, "from": true, "chat": true, "date": true, "text": true, "caption": true, "reply_to_message": true,
	"photo": true, "document": true, "video": true, "audio": true, "voice": true, "video_note": true, "animation": true,
	"sticker": true, "location": true, "venue": true, "contact": true, "poll": true,
	"entities": true, "caption_entities": true, "edit_date": true, "sender_chat": true, "forward_origin": true,
	"forward_from": true, "forward_from_chat": true, "forward_date": true, "forward_signature": true, "forward_sender_name": true,
	"forward_from_message_id": true, "is_automatic_forward": true, "via_bot": true, "media_group_id": true,
	"author_signature": true, "has_protected_content": true, "link_preview_options": true, "reply_markup": true,
	"message_thread_id": true, "is_topic_message": true, "has_media_spoiler": true, "show_caption_above_media": true,
	"external_reply": true, "quote": true, "reply_to_story": true, "business_connection_id": true, "sender_business_bot": true,
	"effect_id": true, "paid_star_count": true, "direct_messages_topic": true, "suggested_post_info": true,
}

func unknownTelegramField(raw []byte) string {
	var u map[string]json.RawMessage
	if json.Unmarshal(raw, &u) != nil {
		return ""
	}
	for _, key := range []string{"message", "edited_message"} {
		var m map[string]json.RawMessage
		if json.Unmarshal(u[key], &m) != nil {
			continue
		}
		for k, v := range m {
			if !telegramKnownFields[k] && string(v) != "null" {
				return k
			}
		}
	}
	return ""
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
	logRawUpdate(raw)

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
	var rich map[string]interface{}

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
	} else if msg.VideoNote != nil {
		mediaType, fileID, mimeType = "video", msg.VideoNote.FileID, "video/mp4"
		rich = map[string]interface{}{"round": true}
	} else if msg.Animation != nil {
		mediaType, fileID, mimeType = "video", msg.Animation.FileID, "video/mp4"
	} else if msg.Sticker != nil && !msg.Sticker.IsAnimated && !msg.Sticker.IsVideo {
		mediaType, fileID, mimeType = "sticker", msg.Sticker.FileID, "image/webp"
	} else if msg.Location != nil {
		mediaType = "location"
		body = "|" + fmt.Sprintf("https://maps.google.com/maps?q=%f%%2C%f&z=17&hl=pt-BR", msg.Location.Latitude, msg.Location.Longitude) + "|"
		if msg.Venue != nil {
			body += strings.ReplaceAll(strings.TrimSpace(msg.Venue.Title+"\n"+msg.Venue.Address), "|", "/")
		}
		if msg.Location.LivePeriod > 0 {
			rich = map[string]interface{}{"live": true, "livePeriod": msg.Location.LivePeriod}
			if body == "|"+fmt.Sprintf("https://maps.google.com/maps?q=%f%%2C%f&z=17&hl=pt-BR", msg.Location.Latitude, msg.Location.Longitude)+"|" {
				body += "Localização em tempo real"
			}
		}
	} else if msg.Contact != nil {
		mediaType = "vcard"
		name := strings.TrimSpace(msg.Contact.FirstName + " " + msg.Contact.LastName)
		body = "BEGIN:VCARD\nVERSION:3.0\nFN:" + name + "\nTEL;type=CELL:+" + strings.TrimPrefix(msg.Contact.PhoneNumber, "+") + "\nEND:VCARD"
	} else if msg.Poll != nil {
		mediaType = "poll"
		lines := []string{"📊 " + msg.Poll.Question}
		for _, o := range msg.Poll.Options {
			lines = append(lines, "• "+o.Text)
		}
		body = strings.Join(lines, "\n")
		opts := make([]string, 0, len(msg.Poll.Options))
		for _, o := range msg.Poll.Options {
			opts = append(opts, o.Text)
		}
		sel := 1
		if msg.Poll.AllowsMultiple {
			sel = len(opts)
		}
		rich = map[string]interface{}{"question": msg.Poll.Question, "options": opts, "selectable": sel}
	} else if msg.Sticker != nil {
		mediaType, body = "unsupported", "sticker animado"
	} else if k := unknownTelegramField(raw); k != "" {
		mediaType, body = "unsupported", k
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
		Edited:          update.EditedMessage != nil && update.Message == nil,
		Forwarded:       len(msg.ForwardOrigin) > 0 && string(msg.ForwardOrigin) != "null",
		Rich:            rich,
	}

	return []domain.InboundMessage{inbound}, nil
}

// logRawUpdate despeja o update cru quando ele não é só texto, para projetar o suporte a tipos novos com o conteúdo
// real. Desligado por padrão: pode ter dado pessoal (WATINK_LOG_CHANNEL_RAW=1 só em diagnóstico).
func logRawUpdate(raw []byte) {
	if os.Getenv("WATINK_LOG_CHANNEL_RAW") != "1" {
		return
	}
	s := string(raw)
	if len(s) > 4000 {
		s = s[:4000] + "…"
	}
	log.Printf("[Telegram] RAW %s", s)
}
