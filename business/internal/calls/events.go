package calls

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
	"strconv"
	"strings"
)

// Estados do registro da chamada.
const (
	StatusRinging     = "ringing"
	StatusActive      = "active"
	StatusEnded       = "ended"
	StatusMissed      = "missed"
	StatusRejected    = "rejected"
	StatusFailed      = "failed"
	StatusInterrupted = "interrupted"
)

// IncomingEvent é o corpo de call.incoming e call.missed publicado pelo engine.
type IncomingEvent struct {
	CallID    string `json:"callId"`
	SessionID string `json:"sessionId"`
	Peer      string `json:"peer"`
	CallerPn  string `json:"callerPn"`
	Direction string `json:"direction"`
	Reason    string `json:"reason"`
}

// StateEvent é o corpo de call.state.
type StateEvent struct {
	CallID    string `json:"callId"`
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	Direction string `json:"direction"`
}

// EndedEvent é o corpo de call.ended.
type EndedEvent struct {
	CallID       string `json:"callId"`
	SessionID    string `json:"sessionId"`
	Peer         string `json:"peer"`
	CallerPn     string `json:"callerPn"`
	Direction    string `json:"direction"`
	EndReason    string `json:"endReason"`
	DurationSecs int    `json:"durationSecs"`
}

// QualityEvent é o corpo de call.quality.
type QualityEvent struct {
	CallID      string   `json:"callId"`
	SessionID   string   `json:"sessionId"`
	RttMs       *float64 `json:"rttMs"`
	LossPct     float64  `json:"lossPct"`
	JitterMs    float64  `json:"jitterMs"`
	NoPeerAudio bool     `json:"noPeerAudio"`
	SilentMs    int64    `json:"silentMs"`
}

func parseSession(raw string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(raw))
	return n
}

// ResolveContactAndTicket acha o contato do chamador (unificando LID e telefone)
// e o ticket aberto da conexão; sem ticket aberto, cria um pendente. Devolve o
// ticket para o registro da chamada aparecer no histórico da conversa.
func (s *Service) ResolveContactAndTicket(ctx context.Context, tenantID uuid.UUID, whatsappID int, peer, callerPn string) (*domain.Contact, *domain.Ticket, error) {
	isLID := strings.HasSuffix(peer, "@lid")
	number := callerPn
	if number == "" {
		number = peerNumber(peer)
	}
	contact, err := s.contacts.FindOrCreate(ctx, tenantID, number, "", "", false, isLID && callerPn == "", peer)
	if err != nil {
		return nil, nil, err
	}
	ticket, err := s.tickets.FindOpenByContact(ctx, tenantID, contact.ID, whatsappID)
	if err != nil {
		return nil, nil, err
	}
	if ticket == nil {
		var queueID *int
		if s.queues != nil {
			if ids, qerr := s.queues.FindQueueIDsByChannel(ctx, whatsappID, tenantID); qerr == nil && len(ids) == 1 {
				queueID = &ids[0]
			}
		}
		ticket, err = s.tickets.FindOrCreatePending(ctx, &domain.Ticket{
			ContactID: contact.ID, Status: "pending", TenantID: tenantID, WhatsappID: whatsappID, QueueID: queueID,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return contact, ticket, nil
}

// upsertLog grava a chamada pela chave (tenantId, callId). Reentrega do mesmo
// evento não duplica: o conflito não faz nada e o registro existente é devolvido.
func (s *Service) upsertLog(tenantID uuid.UUID, l *models.CallLog) (*models.CallLog, bool, error) {
	l.TenantID = tenantID
	res := s.fresh().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenantId"}, {Name: "callId"}},
		DoNothing: true,
	}).Create(l)
	if res.Error != nil {
		return nil, false, res.Error
	}
	created := res.RowsAffected > 0
	var got models.CallLog
	if err := s.fresh().Where(`"tenantId" = ? AND "callId" = ?`, tenantID, l.CallID).First(&got).Error; err != nil {
		return nil, false, err
	}
	return &got, created, nil
}

// HandleIncoming trata call.incoming: registra a chamada, decide quem toca e
// confirma ao engine (call.ready) só se houver alguém para atender. Sem elegível
// não confirma: o engine não envia nada e o celular segue tocando.
func (s *Service) HandleIncoming(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev IncomingEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	waID := parseSession(ev.SessionID)
	if ev.CallID == "" || waID == 0 {
		return fmt.Errorf("call.incoming sem callId/sessionId")
	}
	contact, ticket, err := s.ResolveContactAndTicket(ctx, tenantID, waID, ev.Peer, ev.CallerPn)
	if err != nil {
		return err
	}
	l, created, err := s.upsertLog(tenantID, &models.CallLog{
		CallID: ev.CallID, WhatsappID: waID, ContactID: &contact.ID, TicketID: &ticket.ID,
		Direction: "incoming", Status: StatusRinging, PeerJid: ev.Peer, CallerPn: ev.CallerPn, StartedAt: s.now(),
	})
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	hasPerm := func(userID int) bool { return s.userCan(tenantID, userID, "receive") }
	users, err := s.Eligible(tenantID, waID, hasPerm)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return s.markMissed(ctx, tenantID, l, "no_operator")
	}
	if err := s.command(tenantID, waID, "call.ready", map[string]interface{}{"callId": ev.CallID}); err != nil {
		return err
	}
	payload := s.callPayload(l, contact, ticket)
	for _, u := range users {
		s.bcast.EmitToRoom("/", UserRoom(tenantID, u.ID), "call.incoming", payload)
	}
	return nil
}

func (s *Service) userCan(tenantID uuid.UUID, userID int, action string) bool {
	return auth.UserHasPermission(s.db, userID, tenantID, "calls", action)
}

func (s *Service) command(tenantID uuid.UUID, whatsappID int, cmd string, payload map[string]interface{}) error {
	body := map[string]interface{}{
		"id": uuid.New().String(), "timestamp": s.now().UnixMilli(),
		"tenantId": tenantID.String(), "type": cmd, "payload": payload,
	}
	return s.publisher.PublishCommand(fmt.Sprintf("wbot.%s.%d.%s", tenantID, whatsappID, cmd), body)
}

func (s *Service) callPayload(l *models.CallLog, c *domain.Contact, t *domain.Ticket) map[string]interface{} {
	return map[string]interface{}{
		"callId": l.CallID, "whatsappId": l.WhatsappID, "direction": l.Direction, "status": l.Status,
		"contact": c, "ticketId": l.TicketID, "startedAt": l.StartedAt,
	}
}

// HandleMissed trata call.missed (oferta ignorada pelo engine).
func (s *Service) HandleMissed(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev IncomingEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	waID := parseSession(ev.SessionID)
	if ev.CallID == "" || waID == 0 {
		return fmt.Errorf("call.missed sem callId/sessionId")
	}
	contact, ticket, err := s.ResolveContactAndTicket(ctx, tenantID, waID, ev.Peer, ev.CallerPn)
	if err != nil {
		return err
	}
	l, _, err := s.upsertLog(tenantID, &models.CallLog{
		CallID: ev.CallID, WhatsappID: waID, ContactID: &contact.ID, TicketID: &ticket.ID,
		Direction: "incoming", Status: StatusMissed, PeerJid: ev.Peer, CallerPn: ev.CallerPn,
		StartedAt: s.now(), EndReason: ev.Reason,
	})
	if err != nil {
		return err
	}
	if l.Status == StatusMissed && l.EndedAt != nil {
		return nil
	}
	return s.markMissed(ctx, tenantID, l, ev.Reason)
}

// markMissed encerra o registro como perdido, escreve a mensagem de sistema no
// ticket e avisa os operadores que o toque acabou. Só age uma vez por chamada.
func (s *Service) markMissed(ctx context.Context, tenantID uuid.UUID, l *models.CallLog, reason string) error {
	now := s.now()
	res := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND "endedAt" IS NULL`, tenantID, l.CallID).
		Updates(map[string]interface{}{"status": StatusMissed, "endReason": reason, "endedAt": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	l.Status, l.EndReason, l.EndedAt = StatusMissed, reason, &now
	s.writeSystemMessage(ctx, tenantID, l)
	s.notifyEnded(tenantID, l)
	return nil
}

// notifyEnded manda call.ended ao tenant para o toque sumir em todo navegador.
func (s *Service) notifyEnded(tenantID uuid.UUID, l *models.CallLog) {
	s.bcast.EmitToTenantRoom(tenantID.String(), "call.ended", map[string]interface{}{
		"callId": l.CallID, "status": l.Status, "endReason": l.EndReason,
		"durationSec": l.DurationSec, "ticketId": l.TicketID,
	})
}
