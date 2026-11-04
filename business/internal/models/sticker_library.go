package models

import (
	"time"

	"github.com/google/uuid"
)

// StickerLibraryItem é uma figurinha guardada na biblioteca da EMPRESA: todos os atendentes e todas as conexões WhatsApp dela
// enxergam as mesmas. O arquivo é o WebP final (512x512), o mesmo que o WhatsApp entrega ao destinatário, e fica no
// armazenamento de mídia pelo hash do conteúdo, então salvar a mesma figurinha de novo não duplica (índice único por empresa e
// hash). Não há limite de itens por empresa (decisão do usuário).
type StickerLibraryItem struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	TenantID  uuid.UUID `gorm:"column:tenantId;type:uuid;not null;uniqueIndex:idx_sticker_library_tenant_hash,priority:1" json:"tenantId"`
	Hash      string    `gorm:"column:hash;not null;uniqueIndex:idx_sticker_library_tenant_hash,priority:2" json:"hash"`
	MediaUrl  string    `gorm:"column:mediaUrl;not null" json:"mediaUrl"`
	Width     int       `gorm:"column:width" json:"width"`
	Height    int       `gorm:"column:height" json:"height"`
	Animated  bool      `gorm:"column:animated;not null;default:false" json:"animated"`
	SavedBy   *int      `gorm:"column:savedBy" json:"savedBy"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
}

func (StickerLibraryItem) TableName() string {
	return "StickerLibrary"
}
