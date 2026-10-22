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
	CreatedAt       time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Queues  []Queue  `gorm:"many2many:channel_connection_queues;" json:"queues,omitempty"`
	Tickets []Ticket `gorm:"foreignKey:ChannelConnectionID" json:"tickets,omitempty"`
}

func (ChannelConnection) TableName() string {
	return "ChannelConnections"
}
