package models

import "time"

// Modos da versão do cliente WhatsApp anunciada pelo engine.
const (
	WAVersionModeAuto   = "auto"
	WAVersionModeManual = "manual"
)

// WAVersionSetting é a configuração global (uma linha, id=1, fora do RLS) da versão do cliente WhatsApp que o engine anuncia
// no handshake. Em "auto" o engine resolve sozinho (site oficial, registro wa-version, embutida); em "manual" usa só
// ManualVersion. O papel do engine tem SELECT nesta tabela.
type WAVersionSetting struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	Mode          string    `gorm:"column:mode;not null;default:auto" json:"mode"`
	ManualVersion *string   `gorm:"column:manualVersion" json:"manualVersion"`
	UpdatedAt     time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

func (WAVersionSetting) TableName() string {
	return "WaVersionSettings"
}
