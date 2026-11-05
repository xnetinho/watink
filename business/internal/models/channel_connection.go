package models

import (
	"time"

	"github.com/google/uuid"
)

// ChannelConnection representa uma conexão duradoura de um tenant a um canal específico
// (WhatsApp, Telegram, Mercado Livre, OLX, Instagram, TikTok), com credenciais cifradas
// e status de operação (ADR 0033). Generaliza a antiga entidade Whatsapp.
type ChannelConnection struct {
	ID              int       `gorm:"primaryKey" json:"id"`
	TenantID        uuid.UUID `gorm:"column:tenantId;type:uuid;not null;uniqueIndex:idx_channel_connections_tenant_name" json:"tenantId"`
	Kind            string    `gorm:"column:kind;not null;size:50" json:"kind"`
	Name            string    `gorm:"column:name;not null;size:100;uniqueIndex:idx_channel_connections_tenant_name" json:"name"`
	Status          string    `gorm:"column:status;not null;default:'DISCONNECTED'" json:"status"`
	IsDefault       bool      `gorm:"column:isDefault;not null;default:false" json:"isDefault"`
	CredentialsEnc  string    `gorm:"column:credentialsEnc;type:text" json:"-"`
	Config          string    `gorm:"column:config;type:jsonb;default:'{}'" json:"config"`
	WebhookSecret   string    `gorm:"column:webhookSecret;size:128" json:"-"`
	GreetingMessage string    `gorm:"column:greetingMessage" json:"greetingMessage"`
	FarewellMessage string    `gorm:"column:farewellMessage" json:"farewellMessage"`
	LastError       string    `gorm:"column:lastError;type:text" json:"lastError,omitempty"`
	// Identidade da conta que o canal representa (todo canal tem): o número no WhatsApp, o @usuario no Telegram, o nome de
	// exibição e a foto. Colunas e não JSON: a tela as lê direto e dá para indexar e consultar.
	Identifier  string `gorm:"column:identifier;size:150;not null;default:''" json:"identifier"`
	DisplayName string `gorm:"column:displayName;size:150;not null;default:''" json:"displayName"`
	AvatarURL   string `gorm:"column:avatarUrl;type:text;not null;default:''" json:"avatarUrl"`
	// Estado da sessão (todo canal tem).
	FirstConnectedAt *time.Time `gorm:"column:firstConnectedAt" json:"firstConnectedAt"`
	LastConnectedAt  *time.Time `gorm:"column:lastConnectedAt" json:"lastConnectedAt"`
	CreatedAt        time.Time  `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Queues  []Queue  `gorm:"many2many:channel_connection_queues;" json:"queues,omitempty"`
	Tickets []Ticket `gorm:"foreignKey:ChannelConnectionID" json:"tickets,omitempty"`
}

func (ChannelConnection) TableName() string {
	return "ChannelConnections"
}
