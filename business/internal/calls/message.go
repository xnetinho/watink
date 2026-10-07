package calls

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

// callMessageID é o id da mensagem de sistema de uma chamada: determinístico,
// para a reentrega do mesmo evento nunca criar uma segunda mensagem.
func callMessageID(callID string) string { return "call:" + callID }

// callBody é o texto curto que aparece no preview do ticket.
func callBody(l *models.CallLog) string {
	kind := "Chamada de voz"
	if l.Media == "video" {
		kind = "Videochamada"
	}
	switch l.Status {
	case StatusMissed:
		return kind + " perdida"
	case StatusRejected:
		return kind + " recusada"
	case StatusFailed, StatusInterrupted:
		return kind + " interrompida"
	}
	if l.Direction == "outgoing" {
		return kind + " realizada"
	}
	return kind + " recebida"
}

// writeSystemMessage grava no ticket a mensagem de sistema da chamada
// (mediaType "call") e emite appMessage para a conversa aparecer em tempo real.
// Best-effort: falhar aqui nunca desfaz o registro da chamada.
func (s *Service) writeSystemMessage(ctx context.Context, tenantID uuid.UUID, l *models.CallLog) {
	if l.TicketID == nil {
		return
	}
	payload := map[string]interface{}{
		"callId": l.CallID, "direction": l.Direction, "media": mediaOf(l.Media), "status": l.Status, "endReason": l.EndReason,
		"durationSec": l.DurationSec, "recordingStatus": l.RecordingStatus, "mosEstimated": l.MosEstimated,
	}
	if l.HandledByUserID != nil {
		payload["handledByUserId"] = *l.HandledByUserID
		var u models.User
		if s.fresh().Select("name").Where(`id = ? AND "tenantId" = ?`, *l.HandledByUserID, tenantID).First(&u).Error == nil {
			payload["handledByName"] = u.Name
		}
	}
	data, _ := json.Marshal(payload)
	msg := models.Message{
		ID: callMessageID(l.CallID), Body: callBody(l), TicketID: *l.TicketID, ContactID: l.ContactID,
		FromMe: l.Direction == "outgoing", TenantID: tenantID, MediaType: "call", DataJson: string(data),
	}
	res := s.fresh().Where(`id = ?`, msg.ID).FirstOrCreate(&msg)
	if res.Error != nil {
		log.Printf("[calls] mensagem de sistema da chamada %s falhou: %v", l.CallID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		return
	}
	_ = s.tickets
	ev := map[string]interface{}{"action": "create", "message": msg}
	s.bcast.EmitToRoom("/", fmt.Sprintf("chat:%d", *l.TicketID), "appMessage", ev)
	s.bcast.EmitToTenantRoom(tenantID.String(), "appMessage", ev)
}
