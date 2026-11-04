package services

import (
	"context"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReceiptRow é o "Dados da mensagem" de UM participante: quando recebeu, leu e ouviu (nil = ainda não). Em conversa
// individual há uma linha só (sem participante); em grupo, uma por membro que mandou recibo.
type ReceiptRow struct {
	SenderJID string `json:"senderJid"`
	// Name é o nome do contato quando o JID do participante casa com um contato da empresa; vazio senão.
	Name      string  `json:"name"`
	Delivered *string `json:"delivered"`
	Read      *string `json:"read"`
	Played    *string `json:"played"`
}

const receiptTimeLayout = "2006-01-02T15:04:05Z07:00"

// MessageReceipts devolve os recibos da mensagem agrupados por participante, só da empresa informada.
func MessageReceipts(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, messageID string) ([]ReceiptRow, error) {
	var recs []models.MessageReceipt
	if err := db.WithContext(ctx).
		Where(`"messageId" = ? AND "tenantId" = ?`, messageID, tenantID).
		Order(`"at" ASC`).Find(&recs).Error; err != nil {
		return nil, err
	}
	byJID := map[string]*ReceiptRow{}
	var order []string
	for _, r := range recs {
		row, ok := byJID[r.SenderJID]
		if !ok {
			row = &ReceiptRow{SenderJID: r.SenderJID}
			byJID[r.SenderJID] = row
			order = append(order, r.SenderJID)
		}
		at := r.At.UTC().Format(receiptTimeLayout)
		switch r.Stage {
		case models.ReceiptDelivered:
			row.Delivered = &at
		case models.ReceiptRead:
			row.Read = &at
		case models.ReceiptPlayed:
			row.Played = &at
		}
	}
	out := make([]ReceiptRow, 0, len(order))
	for _, j := range order {
		row := byJID[j]
		row.Name = contactNameForJID(ctx, db, tenantID, j)
		out = append(out, *row)
	}
	return out, nil
}

// contactNameForJID acha o nome do contato pelo número ou pelo LID do JID; vazio se não houver (ou se o JID veio vazio).
func contactNameForJID(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, jid string) string {
	if jid == "" {
		return ""
	}
	user := jid
	if i := strings.IndexAny(user, "@:"); i >= 0 {
		user = user[:i]
	}
	var c models.Contact
	err := db.WithContext(ctx).Select("name").
		Where(`"tenantId" = ? AND (number = ? OR lid = ?)`, tenantID, user, user+"@lid").
		Limit(1).Find(&c).Error
	if err != nil {
		return ""
	}
	return c.Name
}
