package calls

import (
	"context"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/recording"
)

// RecordingURLTTL é a validade da URL assinada de escuta: curta, e gerada a cada
// leitura (a chave é o que se guarda, nunca a URL).
const RecordingURLTTL = 5 * time.Minute

// Ações auditadas em CallRecordingAccess.
const (
	AccessListen = "listen"
	AccessDelete = "delete"
)

// RecordingExcluded marca no histórico uma gravação apagada.
const RecordingExcluded = "deleted"

// recordingOf devolve a chamada SE o usuário pode vê-la e ela tem gravação. Fora do
// alcance, de outra empresa ou sem gravação: sempre ErrNotFound, para nunca revelar
// que a gravação existe.
func (s *Service) recordingOf(sc HistoryScope, callID string) (*models.CallLog, error) {
	l, err := s.Get(sc, callID)
	if err != nil {
		return nil, ErrNotFound
	}
	if l.RecordingKey == "" || l.RecordingStatus != recording.StatusReady {
		return nil, ErrNotFound
	}
	return l, nil
}

// audit grava quem fez o quê e quando. O registro é parte do ato: se falhar, o ato
// não acontece (a escuta não entrega a URL; a exclusão não apaga).
func (s *Service) audit(sc HistoryScope, callID, action string) error {
	return s.fresh().Create(&models.CallRecordingAccess{
		TenantID: sc.TenantID, CallID: callID, UserID: sc.UserID, Action: action, At: s.now(),
	}).Error
}

// ListenURL devolve uma URL assinada e temporária da gravação e registra a escuta.
func (s *Service) ListenURL(ctx context.Context, sc HistoryScope, callID string) (string, error) {
	l, err := s.recordingOf(sc, callID)
	if err != nil {
		return "", err
	}
	if !s.rec.Available() {
		return "", ErrRecordingNoS3
	}
	if err := s.audit(sc, callID, AccessListen); err != nil {
		return "", err
	}
	return s.rec.store.PresignedGetURL(ctx, l.RecordingKey, RecordingURLTTL)
}

// DeleteRecording remove o arquivo do armazenamento, marca o histórico como
// excluído e registra quem excluiu. Excluir duas vezes devolve ErrNotFound.
func (s *Service) DeleteRecording(ctx context.Context, sc HistoryScope, callID string) error {
	l, err := s.recordingOf(sc, callID)
	if err != nil {
		return err
	}
	if !s.rec.Available() {
		return ErrRecordingNoS3
	}
	if err := s.audit(sc, callID, AccessDelete); err != nil {
		return err
	}
	if err := s.rec.store.Delete(ctx, l.RecordingKey); err != nil {
		return err
	}
	return s.fresh().Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ?`, sc.TenantID, callID).
		Updates(map[string]interface{}{"recordingKey": "", "recordingStatus": RecordingExcluded}).Error
}
