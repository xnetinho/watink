package models

import "time"

// WhatsAppConnectionDetails stores WhatsApp-only state without polluting the generic connection row.
type WhatsAppConnectionDetails struct {
	ConnectionID    int        `gorm:"primaryKey;column:connectionId" json:"connectionId"`
	ProxyMode       string     `gorm:"column:proxyMode;default:'none'" json:"proxyMode"`
	ProxyID         *int       `gorm:"column:proxyId" json:"proxyId"`
	ProxyGroupID    *int       `gorm:"column:proxyGroupId" json:"proxyGroupId"`
	SyncHistory     bool       `gorm:"column:syncHistory;default:false" json:"syncHistory"`
	SyncPeriod      string     `gorm:"column:syncPeriod" json:"syncPeriod"`
	KeepAlive       bool       `gorm:"column:keepAlive;default:false" json:"keepAlive"`
	WID             string     `gorm:"column:wid;size:100" json:"wid"`
	LastRiskCode    int        `gorm:"column:lastRiskCode;default:0" json:"lastRiskCode"`
	LastRiskAction  string     `gorm:"column:lastRiskAction" json:"lastRiskAction"`
	LastRiskMessage string     `gorm:"column:lastRiskMessage" json:"lastRiskMessage"`
	LastRiskAt      *time.Time `gorm:"column:lastRiskAt" json:"lastRiskAt"`
}

func (WhatsAppConnectionDetails) TableName() string { return "whatsapp_connection_details" }
