package calls

import (
	"context"
	"fmt"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

// newCallID gera o id de uma chamada originada: 32 hex maiúsculos, o mesmo
// formato que o engine valida.
func newCallID() string {
	return strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
}

// Place origina uma chamada de voz do operador para o contato do ticket. O
// registro nasce ANTES do comando ao engine (assim o call.state/ended que voltar
// já encontra a chamada) e já assumido pelo operador que ligou. Se o comando
// falhar, o registro é fechado como falho e o erro volta ao chamador.
//
// As validações de ticket individual, conexão conectada e sem proxy são do
// controller; aqui ficam a de operador ocupado e a da conexão ocupada.
func (s *Service) Place(ctx context.Context, tenantID uuid.UUID, userID int, ticket *models.Ticket, contact *models.Contact) (*models.CallLog, error) {
	busy, err := s.userInCall(tenantID, userID)
	if err != nil {
		return nil, err
	}
	if busy {
		return nil, ErrUserBusy
	}
	var onConn int64
	if err := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "whatsappId" = ? AND "endedAt" IS NULL AND status IN ?`,
			tenantID, ticket.WhatsappID, []string{StatusRinging, StatusActive}).Count(&onConn).Error; err != nil {
		return nil, err
	}
	if onConn > 0 {
		return nil, ErrConnectionBusy
	}

	number := contact.Number
	if number == "" {
		return nil, fmt.Errorf("contato sem número")
	}
	peer := number + "@s.whatsapp.net"
	now := s.now()
	callID := newCallID()
	l := &models.CallLog{
		TenantID: tenantID, CallID: callID, WhatsappID: ticket.WhatsappID, ContactID: &contact.ID, TicketID: &ticket.ID,
		Direction: "outgoing", Status: StatusRinging, PeerJid: peer, CallerPn: number, StartedAt: now,
		HandledByUserID: &userID, AnsweredAt: nil,
	}
	if err := s.fresh().Create(l).Error; err != nil {
		return nil, err
	}
	if err := s.command(tenantID, ticket.WhatsappID, "call.start", map[string]interface{}{"callId": callID, "to": peer}); err != nil {
		end := s.now()
		s.fresh().Model(&models.CallLog{}).Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).
			Updates(map[string]interface{}{"status": StatusFailed, "endReason": "failed", "endedAt": end})
		return nil, fmt.Errorf("falha ao iniciar a chamada: %w", err)
	}
	return l, nil
}
