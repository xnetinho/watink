package models

import (
	"time"

	"github.com/google/uuid"
)

// MessageEdit é uma versão do texto de uma mensagem. A versão 1 é sempre o texto original; cada edição (do operador ou
// do contato) acrescenta a seguinte. A mensagem guarda só o texto atual (Messages.body) e o contador (editCount): a
// cadeia inteira mora aqui.
//
// (messageId, version) é único: o engine entrega eventos ao menos uma vez e o mesmo evento de edição pode chegar duas
// vezes. O banco, e não só o código, impede a versão duplicada.
type MessageEdit struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	TenantID  uuid.UUID `gorm:"column:tenantId;type:uuid;not null" json:"-"`
	MessageID string    `gorm:"column:messageId;not null;uniqueIndex:uidx_message_edits_message_version,priority:1" json:"messageId"`
	Version   int       `gorm:"not null;uniqueIndex:uidx_message_edits_message_version,priority:2" json:"version"`
	Body      string    `gorm:"not null" json:"body"`
	// EditedAt é o instante da edição segundo o WhatsApp/Telegram (não o da gravação aqui): é ele que ordena a cadeia
	// quando as edições chegam fora de ordem.
	EditedAt time.Time `gorm:"column:editedAt;not null" json:"editedAt"`
	// EditedBy: "me" (operador, pelo Watink) ou "contact" (o contato editou).
	EditedBy  string    `gorm:"column:editedBy;not null;default:'contact'" json:"editedBy"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
}

func (MessageEdit) TableName() string { return "MessageEdits" }

const (
	EditedByMe      = "me"
	EditedByContact = "contact"
)
