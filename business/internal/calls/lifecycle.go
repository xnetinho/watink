package calls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

var (
	ErrAlreadyAnswered = errors.New("a chamada já foi atendida")
	ErrNotFound        = errors.New("chamada não encontrada")
	ErrNotActive       = errors.New("a chamada não está em andamento")
	ErrUserBusy        = errors.New("o operador já está em outra chamada")
)

func (s *Service) load(tenantID uuid.UUID, callID string) (*models.CallLog, error) {
	var l models.CallLog
	err := s.fresh().Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).First(&l).Error
	if err != nil {
		return nil, ErrNotFound
	}
	return &l, nil
}

// userInCall diz se o operador já tem uma chamada ativa (assumida e não encerrada).
func (s *Service) userInCall(tenantID uuid.UUID, userID int) (bool, error) {
	var n int64
	err := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "handledByUserId" = ? AND status IN ?`, tenantID, userID, []string{StatusRinging, StatusActive}).
		Count(&n).Error
	return n > 0, err
}

// Accept atribui a chamada ao operador. A atribuição é um UPDATE condicional com
// checagem de RowsAffected: com dois atendimentos ao mesmo tempo, só o primeiro
// vence e o segundo recebe ErrAlreadyAnswered. Só então o engine é mandado
// atender.
func (s *Service) Accept(ctx context.Context, tenantID uuid.UUID, userID int, callID string) (*models.CallLog, error) {
	l, err := s.load(tenantID, callID)
	if err != nil {
		return nil, err
	}
	busy, err := s.userInCall(tenantID, userID)
	if err != nil {
		return nil, err
	}
	if busy {
		return nil, ErrUserBusy
	}
	now := s.now()
	res := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND status = ? AND "handledByUserId" IS NULL AND direction = 'incoming'`,
			tenantID, callID, StatusRinging).
		Updates(map[string]interface{}{"handledByUserId": userID, "answeredAt": now})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrAlreadyAnswered
	}
	if err := s.command(tenantID, l.WhatsappID, "call.accept", map[string]interface{}{"callId": callID}); err != nil {
		s.fresh().Model(&models.CallLog{}).
			Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).
			Updates(map[string]interface{}{"handledByUserId": nil, "answeredAt": nil})
		return nil, fmt.Errorf("falha ao atender: %w", err)
	}
	s.bcast.EmitToTenantRoom(tenantID.String(), "call.answered", map[string]interface{}{
		"callId": callID, "handledByUserId": userID,
	})
	return s.load(tenantID, callID)
}

// Reject recusa uma chamada que ainda toca. Só aqui o engine envia reject ao chamador.
func (s *Service) Reject(ctx context.Context, tenantID uuid.UUID, userID int, callID string) error {
	l, err := s.load(tenantID, callID)
	if err != nil {
		return err
	}
	if l.Status != StatusRinging || l.HandledByUserID != nil {
		return ErrAlreadyAnswered
	}
	now := s.now()
	res := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND status = ? AND "handledByUserId" IS NULL`, tenantID, callID, StatusRinging).
		Updates(map[string]interface{}{"status": StatusRejected, "endReason": "declined", "endedAt": now, "handledByUserId": userID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadyAnswered
	}
	if err := s.command(tenantID, l.WhatsappID, "call.reject", map[string]interface{}{"callId": callID}); err != nil {
		return fmt.Errorf("falha ao recusar: %w", err)
	}
	l.Status, l.EndReason, l.EndedAt = StatusRejected, "declined", &now
	s.writeSystemMessage(ctx, tenantID, l)
	s.notifyEnded(tenantID, l)
	return nil
}

// End encerra uma chamada em andamento a pedido do operador que a assumiu.
func (s *Service) End(ctx context.Context, tenantID uuid.UUID, userID int, callID string) error {
	l, err := s.load(tenantID, callID)
	if err != nil {
		return err
	}
	if l.EndedAt != nil || l.HandledByUserID == nil || *l.HandledByUserID != userID {
		return ErrNotActive
	}
	return s.command(tenantID, l.WhatsappID, "call.end", map[string]interface{}{"callId": callID})
}

// HandleState trata call.state: a chamada conectou (active).
func (s *Service) HandleState(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev StateEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	if ev.State != "active" {
		return nil
	}
	res := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND status = ? AND "endedAt" IS NULL`, tenantID, ev.CallID, StatusRinging).
		Updates(map[string]interface{}{"status": StatusActive})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		s.bcast.EmitToTenantRoom(tenantID.String(), "call.state", map[string]interface{}{"callId": ev.CallID, "state": "active"})
	}
	return nil
}
