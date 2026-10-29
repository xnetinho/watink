package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name"`
	// number e lid são únicos POR TENANT (idx_contacts_tenant_number e
	// idx_contacts_tenant_lid, criados em database.addCustomIndexes). O unique
	// global antigo (uni_Contacts_number/uni_Contacts_lid) impedia dois tenants
	// de terem o mesmo contato e transformava a duplicata em erro 500.
	Number string `json:"number"`
	// Conexão de origem: todo contato nasce de UMA conexão (importado dela, criado nela, ou vindo de mensagem dela) e
	// ela é a que envia para ele (inclusive no encaminhamento). Mesmo telefone em duas conexões = dois contatos.
	// ConnectionSource diz em qual tabela ConnectionID vive: "whatsapp" (Whatsapps) ou "channel" (ChannelConnections).
	ConnectionSource string `gorm:"column:connectionSource;size:12;not null;default:'whatsapp';index:idx_contacts_connection,priority:1" json:"connectionSource"`
	ConnectionID     int    `gorm:"column:connectionId;not null;default:0;index:idx_contacts_connection,priority:2" json:"connectionId"`
	// PhoneSync: resultado de gravar o contato na agenda do celular da conexão (só WhatsApp): "" (não se aplica),
	// "pending", "saved" ou "failed" (PhoneSyncError explica). O contato existe no Watink em qualquer caso.
	PhoneSync      string `gorm:"column:phoneSync;size:10;not null;default:''" json:"phoneSync"`
	PhoneSyncError string `gorm:"column:phoneSyncError;not null;default:''" json:"phoneSyncError,omitempty"`
	ProfilePicUrl  string `gorm:"column:profilePicUrl" json:"profilePicUrl"`
	Email          string `gorm:"not null;default:''" json:"email"`
	IsGroup        bool   `gorm:"column:isGroup;not null;default:false" json:"isGroup"`
	// GroupParticipantCount é preenchido só para IsGroup=true, via
	// enriquecimento oportunista (plugins/groups_watch.go
	// enrichContactFromGroup) toda vez que o plugin Grupos busca
	// GroupInfo.Participants do WhatsApp -- nunca setado para contato
	// individual. nil = nunca enriquecido (grupo listado só via mensagem
	// recebida, sem passar pelo plugin Grupos ainda).
	GroupParticipantCount *int      `gorm:"column:groupParticipantCount" json:"groupParticipantCount,omitempty"`
	TenantID              uuid.UUID `gorm:"column:tenantId;type:uuid" json:"tenantId"`
	Lid                   *string   `json:"lid"`
	WalletUserID          *int      `gorm:"column:walletUserId" json:"walletUserId"`
	// ClientID is nullable — a Contact can exist and generate Tickets without
	// ever being linked to a Client (someone messages in without being
	// registered). A Contact belongs to at most one Client; the link is
	// always manual (ADR 0023) — no unique constraint here, since multiple
	// Contacts may point to the same Client.
	ClientID  *int      `gorm:"column:clientId" json:"clientId"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Tickets    []Ticket          `gorm:"foreignKey:ContactID" json:"tickets,omitempty"`
	Wallet     *User             `gorm:"foreignKey:WalletUserID" json:"wallet,omitempty"`
	Client     *Client           `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	Identities []ContactIdentity `gorm:"foreignKey:ContactID" json:"identities,omitempty"`
}

func (Contact) TableName() string {
	return "Contacts"
}
