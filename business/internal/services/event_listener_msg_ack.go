package services

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (el *EventListener) handleMessageAck(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	messages := el.messages
	tickets := el.tickets
	var p struct {
		MessageID string `json:"messageId"`
		Ack       int    `json:"ack"`
		// At e By vêm do recibo do WhatsApp: quando e de quem (em grupo, cada participante manda o seu).
		At string `json:"at"`
		By string `json:"by"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	msg, err := messages.FindByID(ctx, p.MessageID, tenantID)
	if err != nil {
		return nil
	}
	if msg == nil {
		// ack for a message we don't have (e.g. the device's own prior messages
		// synced on connect) — nothing to update.
		return nil
	}
	// O recibo é gravado ANTES da regra "o ack só sobe": um recibo de entrega que chega depois do de leitura (fora de ordem)
	// continua sendo um dado verdadeiro e entra no "Dados da mensagem", mesmo sem mudar o ack da mensagem.
	if msg.FromMe {
		recordReceipt(ctx, el.db, tenantID, msg.ID, p.Ack, p.By, p.At)
	}
	// O ack só sobe, com uma exceção: o 5 pode ser um palpite do vigia de entrega (a mensagem não tinha saído do engine
	// no prazo) ou de uma tentativa do engine que falhou e foi repetida. Qualquer confirmação depois dele (1 a 4) prova
	// que a mensagem saiu e desfaz a falha. O ack 1 só é emitido quando o envio deu certo, então não desfaz um erro real.
	if p.Ack > msg.Ack || (msg.Ack == ackFailed && p.Ack >= 1 && p.Ack < ackFailed) {
		if err := messages.Update(ctx, msg, map[string]interface{}{"ack": p.Ack}); err != nil {
			return err
		}
		msg.Ack = p.Ack
		el.bcast().EmitToRoom("/", "chat:"+strconv.Itoa(msg.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": msg})

		// ack >= 3 (read by recipient) on an outgoing message means the contact read our messages.
		// Zero unreadMessages on the ticket and notify the frontend.
		if p.Ack >= 3 && !msg.FromMe {
			ticket, err := tickets.FindByID(ctx, msg.TicketID, tenantID)
			if err == nil && ticket != nil && ticket.UnreadMessages > 0 {
				_ = tickets.Update(ctx, ticket, map[string]interface{}{"unreadMessages": 0})
				ticket.UnreadMessages = 0
				el.bcast().EmitToTenantRoom(tenantID.String(), "ticket", map[string]interface{}{"action": "update", "ticket": ticket})
			}
		}
	}
	return nil
}

// recordReceipt guarda o horário do recibo (entregue/lido/reproduzido) de uma mensagem nossa. É melhor esforço: o ack da
// mensagem já é a verdade de produto, e uma falha aqui não pode impedir a atualização dele. Idempotente: o mesmo recibo
// duas vezes mantém o primeiro horário.
func recordReceipt(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, messageID string, ack int, by, at string) {
	stage, ok := models.ReceiptStageForAck(ack)
	if !ok || db == nil {
		return
	}
	when, err := time.Parse(time.RFC3339, at)
	if err != nil {
		when = time.Now()
	}
	rec := models.MessageReceipt{TenantID: tenantID, MessageID: messageID, SenderJID: by, Stage: stage, At: when}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		log.Printf("[handleMessageAck] recibo %s/%s de %s não gravado: %v", messageID, stage, by, err)
	}
}
