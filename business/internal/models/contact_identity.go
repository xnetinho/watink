package models

import (
	"time"

	"github.com/google/uuid"
)

// ContactIdentity representa a identidade externa de um Contact em um canal específico
// (WhatsApp, Telegram, Mercado Livre, OLX, Instagram, TikTok). Um Contact humano pode possuir N
// identidades em canais diferentes (ADR 0033).
type ContactIdentity struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	TenantID    uuid.UUID `gorm:"column:tenantId;type:uuid;not null;index:idx_contact_identities_tenant;index:idx_contact_identities_tenant_channel_ext,unique" json:"tenantId"`
	ContactID   int       `gorm:"column:contactId;not null;index:idx_contact_identities_contact" json:"contactId"`
	Channel     string    `gorm:"column:channel;not null;index:idx_contact_identities_tenant_channel_ext,unique" json:"channel"`
	ExternalID  string    `gorm:"column:externalId;not null;index:idx_contact_identities_tenant_channel_ext,unique" json:"externalId"`
	DisplayName string    `gorm:"column:displayName;not null;default:''" json:"displayName"`
	AvatarURL   string    `gorm:"column:avatarUrl;not null;default:''" json:"avatarUrl"`
	Meta        string    `gorm:"column:meta;type:jsonb;default:'{}'" json:"meta"`
	CreatedAt   time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Contact *Contact `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
}

func (ContactIdentity) TableName() string {
	return "ContactIdentities"
}
