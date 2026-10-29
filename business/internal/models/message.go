package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID                  string    `gorm:"primaryKey" json:"id"`
	ExternalID          string    `gorm:"column:externalId" json:"externalId"`
	ChannelConnectionID *int      `gorm:"column:channelConnectionId;index:idx_messages_tenant_conn_ext,priority:2" json:"channelConnectionId,omitempty"`
	Body                string    `gorm:"not null" json:"body"`
	Ack                 int       `gorm:"not null;default:0" json:"ack"`
	Read                bool      `gorm:"not null;default:false" json:"read"`
	MediaType           string    `gorm:"column:mediaType" json:"mediaType"`
	MediaUrl            string    `gorm:"column:mediaUrl" json:"mediaUrl"`
	TicketID            int       `gorm:"column:ticketId;not null" json:"ticketId"`
	FromMe              bool      `gorm:"column:fromMe;not null;default:false" json:"fromMe"`
	IsDeleted           bool      `gorm:"column:isDeleted;not null;default:false" json:"isDeleted"`
	ContactID           *int      `gorm:"column:contactId" json:"contactId"`
	QuotedMsgID         *string   `gorm:"column:quotedMsgId" json:"quotedMsgId"`
	TenantID            uuid.UUID `gorm:"column:tenantId;type:uuid;index:idx_messages_tenant_conn_ext,priority:1" json:"tenantId"`
	Reactions   string    `gorm:"type:jsonb;default:'[]'" json:"reactions"`
	DataJson    string    `gorm:"column:dataJson;type:jsonb;default:'{}'" json:"dataJson"`
	Participant string    `json:"participant"`
	// Transcription is filled on-demand when an operator clicks "Transcrever
	// áudio" on an audio message (message_transcription.go) — never written
	// automatically on receipt, and never sent back to WhatsApp.
	Transcription *string   `gorm:"column:transcription" json:"transcription"`
	CreatedAt     time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// EditCount é quantas vezes a mensagem foi editada (0 = nunca). As versões estão em MessageEdits; aqui fica só o
	// contador, para a tela mostrar o selo "Editada" sem buscar a cadeia.
	EditCount int `gorm:"column:editCount;not null;default:0" json:"editCount"`

	// QuotedMsg é a mensagem citada, preenchida só na listagem (GET /messages/:ticketId) para o
	// balão mostrar "em resposta a…". Não é coluna.
	QuotedMsg *Message `gorm:"-" json:"quotedMsg,omitempty"`

	// Relations
	Ticket Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
}

func (Message) TableName() string {
	return "Messages"
}
