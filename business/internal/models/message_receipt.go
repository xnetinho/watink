package models

import (
	"time"

	"github.com/google/uuid"
)

// MessageReceipt é um recibo de uma mensagem nossa: quando o contato recebeu, leu ou ouviu, segundo o WhatsApp. Uma linha
// por mensagem, participante e estágio; em conversa individual há um participante só, em grupo há um por membro. É o que
// alimenta o "Dados da mensagem". A mensagem guarda apenas o estágio mais alto (Messages.ack): o histórico fica aqui.
//
// (messageId, senderJid, stage) é único: o engine entrega eventos ao menos uma vez e o recibo repetido não pode duplicar nem
// mudar o horário já registrado. O banco, e não só o código, impede a duplicata.
type MessageReceipt struct {
	ID        int       `gorm:"primaryKey" json:"-"`
	TenantID  uuid.UUID `gorm:"column:tenantId;type:uuid;not null" json:"-"`
	MessageID string    `gorm:"column:messageId;not null;uniqueIndex:uidx_message_receipts,priority:1" json:"messageId"`
	// SenderJID identifica quem mandou o recibo (JID sem dispositivo). Vazio quando o WhatsApp não informa (conversa individual).
	SenderJID string `gorm:"column:senderJid;not null;default:'';uniqueIndex:uidx_message_receipts,priority:2" json:"senderJid"`
	Stage     string `gorm:"not null;uniqueIndex:uidx_message_receipts,priority:3" json:"stage"`
	// At é o instante do recibo segundo o WhatsApp (não o da gravação).
	At        time.Time `gorm:"column:at;not null" json:"at"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"-"`
}

func (MessageReceipt) TableName() string { return "MessageReceipts" }

const (
	ReceiptDelivered = "delivered"
	ReceiptRead      = "read"
	ReceiptPlayed    = "played"
)

// ReceiptStageForAck converte o ack numérico do engine no estágio do recibo. Ack 1 (enviado ao servidor) e 5 (erro) não
// são recibos do contato.
func ReceiptStageForAck(ack int) (string, bool) {
	switch ack {
	case 2:
		return ReceiptDelivered, true
	case 3:
		return ReceiptRead, true
	case 4:
		return ReceiptPlayed, true
	}
	return "", false
}
