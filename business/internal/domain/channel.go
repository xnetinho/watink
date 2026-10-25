package domain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

// ChannelKind identifica o tipo de canal de mensageria suportado pela plataforma.
type ChannelKind string

const (
	ChannelKindWhatsApp     ChannelKind = "whatsapp"
	ChannelKindTelegram     ChannelKind = "telegram"
	ChannelKindMercadoLivre ChannelKind = "mercadolivre"
	ChannelKindOLX          ChannelKind = "olx"
	ChannelKindInstagram    ChannelKind = "instagram"
	ChannelKindTikTok       ChannelKind = "tiktok"
)

// MediaKind identifica tipos de mídia suportados para envio/recebimento.
type MediaKind string

const (
	MediaKindImage    MediaKind = "image"
	MediaKindVideo    MediaKind = "video"
	MediaKindAudio    MediaKind = "audio"
	MediaKindDocument MediaKind = "document"
	MediaKindSticker  MediaKind = "sticker"
)

// AuthKind define o modelo de autenticação/provisionamento da conexão.
type AuthKind string

const (
	AuthKindQR          AuthKind = "qr"           // QR Code / Pairing Code (WhatsApp)
	AuthKindToken       AuthKind = "token"        // Token estático / API key (Telegram Bot)
	AuthKindOAuth       AuthKind = "oauth"        // Fluxo OAuth padrão (Mercado Livre, OLX)
	AuthKindOAuthReview AuthKind = "oauth_review" // OAuth com aprovação de app pela plataforma (Instagram, TikTok)
)

// ReplyBlockReason indica por que uma resposta está bloqueada no canal.
type ReplyBlockReason string

const (
	ReplyBlockNone                 ReplyBlockReason = "none"
	ReplyBlockWindowExpired        ReplyBlockReason = "reply_window_expired"
	ReplyBlockInitiatorNotAllowed  ReplyBlockReason = "initiator_not_allowed"
	ReplyBlockMessageLimitExceeded ReplyBlockReason = "message_limit_exceeded"
	ReplyBlockDisconnected         ReplyBlockReason = "channel_disconnected"
)

// ReplyWindowExtension descreve tags ou extensões para janelas de resposta (ex: Human Agent no Instagram).
type ReplyWindowExtension struct {
	Tag      string        `json:"tag"`
	Duration time.Duration `json:"duration"`
}

// Capabilities define em dados as capacidades estruturais e restrições de um canal (D1 / ADR 0033).
type Capabilities struct {
	InitiateConversation bool                  `json:"initiateConversation"` // Se a empresa pode iniciar conversa sem contato prévio do cliente
	ReplyWindow          *time.Duration        `json:"replyWindow"`          // Janela máxima de resposta (nil se ilimitada)
	ReplyWindowExtension *ReplyWindowExtension `json:"replyWindowExtension,omitempty"`
	MaxTextBytes         int                   `json:"maxTextBytes"`
	Media                []MediaKind           `json:"media"`
	Groups               bool                  `json:"groups"`
	Typing               bool                  `json:"typing"`
	ReadReceipts         bool                  `json:"readReceipts"`
	Reactions            bool                  `json:"reactions"`
	Voice                bool                  `json:"voice"`
	Video                bool                  `json:"video"`
	Auth                 AuthKind              `json:"auth"`
	HasPhone             bool                  `json:"hasPhone"` // Se o canal garante telefone real do contato
}

// InboundSender carrega os dados de quem enviou a mensagem externa.
type InboundSender struct {
	ExternalID string `json:"externalId"` // Identificador único no canal (JID, chat_id, IGSID, etc.)
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	Phone      string `json:"phone,omitempty"`
	IsGroup    bool   `json:"isGroup"`
	GroupJID   string `json:"groupJid,omitempty"`
}

// InboundMessage é o contrato neutro e universal de mensagem recebida de qualquer canal (D6 / ADR 0033).
type InboundMessage struct {
	TenantID            uuid.UUID     `json:"tenantId"`
	ChannelConnectionID int           `json:"channelConnectionId"`
	Channel             ChannelKind   `json:"channel"`
	ExternalMessageID   string        `json:"externalMessageId"`
	ConversationKey     string        `json:"conversationKey"` // Identificador de agrupamento da conversa (contato, pack_id, ad_id)
	Sender              InboundSender `json:"sender"`
	Body                string        `json:"body"`
	MediaType           string        `json:"mediaType,omitempty"`
	MediaURL            string        `json:"mediaUrl,omitempty"`
	Timestamp           time.Time     `json:"timestamp"`
	QuotedMessageID     string        `json:"quotedMessageId,omitempty"`
	RawPayload          []byte        `json:"-"`
}

// OutboundMessage representa uma mensagem a ser transmitida para a plataforma externa.
type OutboundMessage struct {
	ConversationKey string            `json:"conversationKey"`
	ExternalID      string            `json:"externalId"`
	Body            string            `json:"body"`
	MediaType       string            `json:"mediaType,omitempty"`
	MediaURL        string            `json:"mediaUrl,omitempty"`
	MediaData       []byte            `json:"-"`
	QuotedMessageID string            `json:"quotedMessageId,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// StorageMessageID é o id com que a mensagem é gravada em Messages. O id do canal só é único dentro do seu escopo (no
// Telegram, dentro do chat), então fora do WhatsApp o id inclui canal, conexão e conversa. O WhatsApp mantém o id cru
// porque o engine, as citações e as confirmações de leitura dependem dele.
func StorageMessageID(kind ChannelKind, connectionID int, conversationKey, externalID string) string {
	if kind == ChannelKindWhatsApp {
		return externalID
	}
	return fmt.Sprintf("%s:%d:%s:%s", kind, connectionID, conversationKey, externalID)
}

// SendResult traz os dados de confirmação da entrega ao canal externo.
type SendResult struct {
	ExternalMessageID string    `json:"externalMessageId"`
	Timestamp         time.Time `json:"timestamp"`
	Status            string    `json:"status"`
}

// ConnectInput dados fornecidos para autenticação ou início da conexão.
type ConnectInput struct {
	Token       string            `json:"token,omitempty"`
	OAuthCode   string            `json:"oauthCode,omitempty"`
	RedirectURI string            `json:"redirectUri,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
}

// ConnectResult resultado da tentativa de conexão (status, QR code se aplicável, etc.).
type ConnectResult struct {
	Status    string `json:"status"`
	QRCode    string `json:"qrcode,omitempty"`
	Connected bool   `json:"connected"`
}

// Channel é o contrato neutro e desacoplado que todo adaptador de mensageria implementa (D1 / ADR 0033).
type Channel interface {
	Kind() ChannelKind
	Capabilities() Capabilities
	Connect(ctx context.Context, conn *models.ChannelConnection, input ConnectInput) (ConnectResult, error)
	Disconnect(ctx context.Context, conn *models.ChannelConnection) error
	Send(ctx context.Context, conn *models.ChannelConnection, msg OutboundMessage) (SendResult, error)
	ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error)
	ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]InboundMessage, error)
}

// ChannelRegistry gerencia os adaptadores de canais registrados no sistema.
type ChannelRegistry struct {
	mu       sync.RWMutex
	channels map[ChannelKind]Channel
}

var (
	defaultRegistry     *ChannelRegistry
	defaultRegistryOnce sync.Once
)

// DefaultChannelRegistry retorna o singleton global de registro de canais.
func DefaultChannelRegistry() *ChannelRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewChannelRegistry()
	})
	return defaultRegistry
}

func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{
		channels: make(map[ChannelKind]Channel),
	}
}

func (r *ChannelRegistry) Register(c Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[c.Kind()] = c
}

func (r *ChannelRegistry) Get(kind ChannelKind) (Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.channels[kind]
	if !ok {
		return nil, fmt.Errorf("channel adapter not registered for kind: %s", kind)
	}
	return c, nil
}

func (r *ChannelRegistry) List() []Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Channel, 0, len(r.channels))
	for _, c := range r.channels {
		list = append(list, c)
	}
	return list
}
