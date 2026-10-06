package calls

import (
	"context"
	"encoding/json"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

// statusForEnd traduz o motivo do engine no estado final do registro.
func statusForEnd(l *models.CallLog, reason string) string {
	switch reason {
	case "interrupted":
		return StatusInterrupted
	case "failed":
		return StatusFailed
	case "declined":
		return StatusRejected
	}
	if l.AnsweredAt == nil || l.HandledByUserID == nil {
		return StatusMissed
	}
	return StatusEnded
}

// HandleEnded trata call.ended: fecha o registro com a duração real e o resumo
// de qualidade, escreve a mensagem de sistema no ticket e avisa os navegadores.
// Idempotente: só a primeira entrega fecha (UPDATE condicional em endedAt).
func (s *Service) HandleEnded(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev EndedEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	l, err := s.load(tenantID, ev.CallID)
	if err != nil {
		return nil
	}
	if l.EndedAt != nil {
		return nil
	}
	now := s.now()
	status := statusForEnd(l, ev.EndReason)
	upd := map[string]interface{}{
		"status": status, "endReason": ev.EndReason, "endedAt": now, "durationSec": ev.DurationSecs,
	}
	if sum := s.takeQuality(tenantID, ev.CallID); sum != nil {
		r := sum.Result()
		upd["rttAvg"], upd["rttMax"] = r.RttAvg, r.RttMax
		upd["lossAvg"], upd["lossMax"] = r.LossAvg, r.LossMax
		upd["jitterAvg"], upd["jitterMax"] = r.JitterAvg, r.JitterMax
		upd["mosEstimated"] = r.Mos
		upd["qualitySamples"] = sum.N
	}
	res := s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND "endedAt" IS NULL`, tenantID, ev.CallID).
		Updates(upd)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	done, err := s.load(tenantID, ev.CallID)
	if err != nil {
		return err
	}
	s.finalizeRecording(ctx, tenantID, ev.CallID)
	if again, err := s.load(tenantID, ev.CallID); err == nil {
		done = again
	}
	s.writeSystemMessage(ctx, tenantID, done)
	s.notifyEnded(tenantID, done)
	return nil
}

// HandleReset trata call.reset: o engine (re)iniciou a sessão e não tem nenhuma
// chamada. Toda chamada ainda aberta daquela conexão ficou órfã.
func (s *Service) HandleReset(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	waID := parseSession(ev.SessionID)
	if waID == 0 {
		return nil
	}
	var open []models.CallLog
	if err := s.fresh().
		Where(`"tenantId" = ? AND "whatsappId" = ? AND "endedAt" IS NULL AND status IN ?`,
			tenantID, waID, []string{StatusRinging, StatusActive}).
		Find(&open).Error; err != nil {
		return err
	}
	for i := range open {
		l := open[i]
		now := s.now()
		res := s.fresh().Model(&models.CallLog{}).
			Where(`"tenantId" = ? AND "callId" = ? AND "endedAt" IS NULL`, tenantID, l.CallID).
			Updates(map[string]interface{}{"status": StatusInterrupted, "endReason": "interrupted", "endedAt": now})
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		l.Status, l.EndReason, l.EndedAt = StatusInterrupted, "interrupted", &now
		s.takeQuality(tenantID, l.CallID)
		if s.rec != nil {
			s.rec.Discard(tenantID, l.CallID)
		}
		s.writeSystemMessage(ctx, tenantID, &l)
		s.notifyEnded(tenantID, &l)
	}
	return nil
}

// HandleQuality trata call.quality: interpreta a medição, acumula o resumo e
// entrega SÓ ao operador que assumiu a chamada (nunca à empresa toda).
func (s *Service) HandleQuality(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev QualityEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	l, err := s.load(tenantID, ev.CallID)
	if err != nil || l.EndedAt != nil || l.HandledByUserID == nil {
		return nil
	}
	sample := Sample{RttMs: ev.RttMs, LossPct: ev.LossPct, JitterMs: ev.JitterMs}
	a := Assess(sample)
	s.addQuality(tenantID, ev.CallID, sample)
	s.bcast.EmitToRoom("/", UserRoom(tenantID, *l.HandledByUserID), "call.quality", map[string]interface{}{
		"callId": ev.CallID, "rttMs": ev.RttMs, "lossPct": ev.LossPct, "jitterMs": ev.JitterMs,
		"level": a.Level, "mosEstimated": a.Mos, "alerts": a.Alerts,
		"noPeerAudio": ev.NoPeerAudio, "silentMs": ev.SilentMs,
	})
	return nil
}

func (s *Service) addQuality(tenantID uuid.UUID, callID string, sample Sample) {
	key := tenantID.String() + ":" + callID
	s.qualityMu.Lock()
	defer s.qualityMu.Unlock()
	sum := s.quality[key]
	if sum == nil {
		sum = &Summary{}
		s.quality[key] = sum
	}
	sum.Add(sample)
}

func (s *Service) takeQuality(tenantID uuid.UUID, callID string) *Summary {
	key := tenantID.String() + ":" + callID
	s.qualityMu.Lock()
	defer s.qualityMu.Unlock()
	sum := s.quality[key]
	delete(s.quality, key)
	return sum
}
