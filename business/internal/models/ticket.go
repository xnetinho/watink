package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID                  int       `gorm:"primaryKey" json:"id"`
	Status              string    `gorm:"not null;default:'pending'" json:"status"`
	LastMessage         string    `gorm:"column:lastMessage" json:"lastMessage"`
	// LastMessageType é o marcador de tipo da última mensagem quando ela não tem texto legível (image, video, poll,
	// location...). A prévia do ticket é traduzida pela tela; o banco nunca guarda rótulo em um idioma.
	LastMessageType string `gorm:"column:lastMessageType;size:30" json:"lastMessageType,omitempty"`
	ContactID           int       `gorm:"column:contactId" json:"contactId"`
	UserID              *int      `gorm:"column:userId" json:"userId"`
	WhatsappID          int       `gorm:"column:whatsappId" json:"whatsappId"`
	IsGroup             bool      `gorm:"column:isGroup;not null;default:false" json:"isGroup"`
	IsCommunity         bool      `gorm:"column:isCommunity;not null;default:false" json:"isCommunity"`
	IsSubGroup          bool      `gorm:"column:isSubGroup;not null;default:false" json:"isSubGroup"`
	UnreadMessages      int       `gorm:"column:unreadMessages" json:"unreadMessages"`
	QueueID             *int      `gorm:"column:queueId" json:"queueId"`
	TenantID            uuid.UUID `gorm:"column:tenantId;type:uuid" json:"tenantId"`
	ChannelConnectionID *int      `gorm:"column:channelConnectionId;index:idx_tickets_channel_conn;constraint:OnDelete:CASCADE" json:"channelConnectionId,omitempty"`
	ConversationKey     string    `gorm:"column:conversationKey;size:255;index:idx_tickets_conv_key" json:"conversationKey,omitempty"`
	CreatedAt           time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Contact           Contact            `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
	User              User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ChannelConnection *ChannelConnection `gorm:"foreignKey:ChannelConnectionID;references:ID" json:"channelConnection,omitempty"`
	Messages          []Message          `gorm:"foreignKey:TicketID" json:"messages,omitempty"`

	// Tags is populated manually (batch query via EntityTags) by ListTickets —
	// not a GORM association, since EntityTag is polymorphic (entityType+entityId).
	Tags []Tag `gorm:"-" json:"tags,omitempty"`

	// Channel e Capabilities expostos para o frontend agnóstico (Tarefa 7.3 / ADR 0033)
	Channel string `gorm:"-" json:"channel,omitempty"`
	// Connection identifica a conexão do ticket (canal + nome), neutra para qualquer canal (domain.TicketConnection).
	// Preenchida em lote por attachChannelInfo e em ShowTicket; não é coluna.
	Connection   *TicketConnectionInfo `gorm:"-" json:"connection,omitempty"`
	Capabilities any                   `gorm:"-" json:"capabilities,omitempty"`
}

func (Ticket) TableName() string {
	return "Tickets"
}

// TicketConnectionInfo espelha domain.TicketConnection no modelo (o pacote models não importa domain).
type TicketConnectionInfo struct {
	Kind   string `json:"kind"`
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}
