package models

import "time"

// WhatsAppConnectionDetails stores WhatsApp-only state without polluting the generic connection row.
type WhatsAppConnectionDetails struct {
	ConnectionID int `gorm:"primaryKey;column:connectionId" json:"connectionId"`
	// EngineType é a biblioteca que atende esta conexão (whatsmeow hoje). Mora aqui, e não em ChannelConnections, para que o
	// suporte a outras bibliotecas (zapo, whatsapp-rust) não toque na tabela genérica.
	EngineType string `gorm:"column:engineType;not null;default:'whatsmeow'" json:"engineType"`
	// Presença: quando o número desta conexão fica "disponível" no WhatsApp (nunca / durante o atendimento / sempre).
	PresencePolicy      string `gorm:"column:presencePolicy;not null;default:'never'" json:"presencePolicy"`
	PresenceIdleMinutes int    `gorm:"column:presenceIdleMinutes;not null;default:5" json:"presenceIdleMinutes"`
	// ConnectionGroupID agrupa conexões WhatsApp na tela (ConnectionGroups).
	ConnectionGroupID *int       `gorm:"column:connectionGroupId;index" json:"connectionGroupId"`
	ProxyMode         string     `gorm:"column:proxyMode;default:'none'" json:"proxyMode"`
	ProxyID           *int       `gorm:"column:proxyId" json:"proxyId"`
	ProxyGroupID      *int       `gorm:"column:proxyGroupId" json:"proxyGroupId"`
	SyncHistory       bool       `gorm:"column:syncHistory;default:false" json:"syncHistory"`
	SyncPeriod        string     `gorm:"column:syncPeriod" json:"syncPeriod"`
	KeepAlive         bool       `gorm:"column:keepAlive;default:false" json:"keepAlive"`
	WID               string     `gorm:"column:wid;size:100" json:"wid"`
	LastRiskCode      int        `gorm:"column:lastRiskCode;default:0" json:"lastRiskCode"`
	LastRiskAction    string     `gorm:"column:lastRiskAction" json:"lastRiskAction"`
	LastRiskMessage   string     `gorm:"column:lastRiskMessage" json:"lastRiskMessage"`
	LastRiskAt        *time.Time `gorm:"column:lastRiskAt" json:"lastRiskAt"`
}

func (WhatsAppConnectionDetails) TableName() string { return "whatsapp_connection_details" }
