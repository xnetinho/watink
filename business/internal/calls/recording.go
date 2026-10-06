package calls

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/recording"
	"github.com/google/uuid"
)

var (
	ErrRecordingOff      = errors.New("a gravação de chamadas está desligada nesta empresa")
	ErrRecordingNotOpt   = errors.New("a gravação por chamada só está disponível no modo opcional")
	ErrRecordingActive   = errors.New("a chamada já está sendo gravada")
	ErrRecordingNone     = errors.New("a chamada não está sendo gravada")
	ErrNoRecordingFile   = errors.New("a chamada não tem gravação")
	ErrRecordingNoS3     = recording.ErrNoStorage
	ErrCallNotInProgress = errors.New("a chamada não está em andamento")
)

// Chaves e modos por empresa. Espelham controllers/setting_calls.go (que não pode
// ser importado daqui: controllers já importa calls). Um teste garante que os
// valores continuam iguais.
const (
	SettingRecordingMode  = "callRecordingMode"
	SettingRecordingAckBy = "callRecordingAckBy"
	SettingRecordingAckAt = "callRecordingAckAt"

	CallRecordingOff      = "off"
	CallRecordingOptional = "optional"
	CallRecordingAuto     = "auto"
)

// NormalizeRecordingMode devolve um dos três modos; qualquer outro valor vira "off".
func NormalizeRecordingMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case CallRecordingOptional:
		return CallRecordingOptional
	case CallRecordingAuto:
		return CallRecordingAuto
	}
	return CallRecordingOff
}

// recTickEvery é o passo do relógio do gravador: um quadro de 20 ms.
const recTickEvery = 20 * time.Millisecond

type activeRecording struct {
	rec  *recording.Recorder
	stop chan struct{}
	once sync.Once
}

func (a *activeRecording) halt() { a.once.Do(func() { close(a.stop) }) }

// Recording é o conjunto de gravações em andamento (uma por chamada).
type Recording struct {
	store domain.ObjectStore
	dir   string

	mu   sync.Mutex
	live map[string]*activeRecording
}

func NewRecording(store domain.ObjectStore, tmpDir string) *Recording {
	return &Recording{store: store, dir: tmpDir, live: map[string]*activeRecording{}}
}

// Available diz se a instalação consegue gravar (tem armazenamento de objetos).
func (r *Recording) Available() bool { return r != nil && r.store != nil }

func recKey(tenantID uuid.UUID, callID string) string { return tenantID.String() + ":" + callID }

// Active diz se a chamada está sendo gravada agora.
func (r *Recording) Active(tenantID uuid.UUID, callID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.live[recKey(tenantID, callID)]
	return ok
}

// Start abre o gravador da chamada e liga o relógio de 20 ms. Idempotente por
// chamada: uma segunda chamada devolve ErrRecordingActive.
func (r *Recording) Start(tenantID uuid.UUID, callID string) (*recording.Recorder, error) {
	if !r.Available() {
		return nil, recording.ErrNoStorage
	}
	k := recKey(tenantID, callID)
	r.mu.Lock()
	if _, ok := r.live[k]; ok {
		r.mu.Unlock()
		return nil, ErrRecordingActive
	}
	rec, err := recording.New(r.dir)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	ar := &activeRecording{rec: rec, stop: make(chan struct{})}
	r.live[k] = ar
	r.mu.Unlock()

	go func() {
		t := time.NewTicker(recTickEvery)
		defer t.Stop()
		for {
			select {
			case <-ar.stop:
				return
			case <-t.C:
				if err := rec.Tick(); err != nil {
					return
				}
			}
		}
	}()
	return rec, nil
}

// Feed entrega um quadro ao gravador da chamada, se houver um. Nunca bloqueia.
func (r *Recording) Feed(tenantID uuid.UUID, callID string, fromOperator bool, frame []byte) {
	r.mu.Lock()
	ar := r.live[recKey(tenantID, callID)]
	r.mu.Unlock()
	if ar == nil {
		return
	}
	if fromOperator {
		ar.rec.AddOperator(frame)
	} else {
		ar.rec.AddPeer(frame)
	}
}

// Stop encerra a gravação da chamada, envia o MP3 ao armazenamento e devolve a
// chave e a duração. Sem gravação em curso devolve ErrRecordingNone. O arquivo
// temporário é sempre removido; em falha de envio nada fica exposto.
func (r *Recording) Stop(ctx context.Context, tenantID uuid.UUID, callID string) (key string, dur time.Duration, err error) {
	k := recKey(tenantID, callID)
	r.mu.Lock()
	ar := r.live[k]
	delete(r.live, k)
	r.mu.Unlock()
	if ar == nil {
		return "", 0, ErrRecordingNone
	}
	ar.halt()
	f, d, ferr := ar.rec.Finish()
	if ferr != nil {
		ar.rec.Discard()
		return "", 0, ferr
	}
	key, uerr := recording.Upload(ctx, r.store, tenantID, callID, f)
	if uerr != nil {
		return "", d, uerr
	}
	return key, d, nil
}

// Discard descarta a gravação em curso sem enviar nada.
func (r *Recording) Discard(tenantID uuid.UUID, callID string) {
	k := recKey(tenantID, callID)
	r.mu.Lock()
	ar := r.live[k]
	delete(r.live, k)
	r.mu.Unlock()
	if ar != nil {
		ar.halt()
		ar.rec.Discard()
	}
}

// finalize fecha a gravação ao fim da chamada e registra o resultado no CallLog:
// ready (com chave e duração) ou failed (sem expor áudio parcial).
func (s *Service) finalizeRecording(ctx context.Context, tenantID uuid.UUID, callID string) {
	if s.rec == nil || !s.rec.Active(tenantID, callID) {
		return
	}
	key, dur, err := s.rec.Stop(ctx, tenantID, callID)
	upd := map[string]interface{}{"recordingStatus": recording.StatusReady, "recordingKey": key, "recordingDurationSec": int(dur.Seconds())}
	if err != nil {
		log.Printf("[calls] gravação da chamada %s falhou: %v", callID, err)
		upd = map[string]interface{}{"recordingStatus": recording.StatusFailed, "recordingKey": ""}
	}
	s.fresh().Model(&models.CallLog{}).Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).Updates(upd)
}

// autoRecord diz se a empresa grava todas as chamadas.
func (s *Service) autoRecord(tenantID uuid.UUID) bool {
	return s.rec.Available() && s.recordingMode(tenantID) == CallRecordingAuto
}

// startRecordingBestEffort liga a gravação sem nunca atrapalhar a chamada: se falhar,
// registra e segue sem gravar (o histórico mostra recordingStatus=failed).
func (s *Service) startRecordingBestEffort(tenantID uuid.UUID, userID int, callID string) {
	if err := s.StartRecording(tenantID, userID, callID, true); err != nil && !errors.Is(err, ErrRecordingActive) {
		log.Printf("[calls] gravação automática da chamada %s não iniciou: %v", callID, err)
		s.fresh().Model(&models.CallLog{}).Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).
			Update("recordingStatus", recording.StatusFailed)
	}
}

// recordingMode lê o modo de gravação da empresa; ausente ou inválido = off.
func (s *Service) recordingMode(tenantID uuid.UUID) string {
	var st models.Setting
	if err := s.fresh().Where(`key = ? AND "tenantId" = ?`, SettingRecordingMode, tenantID).First(&st).Error; err != nil {
		return CallRecordingOff
	}
	return NormalizeRecordingMode(st.Value)
}

// StartRecording inicia a gravação de uma chamada em andamento do operador. `auto`
// vale para o disparo automático (modo auto); o pedido manual exige o modo opcional.
func (s *Service) StartRecording(tenantID uuid.UUID, userID int, callID string, auto bool) error {
	if !s.rec.Available() {
		return ErrRecordingNoS3
	}
	mode := s.recordingMode(tenantID)
	switch {
	case mode == CallRecordingOff:
		return ErrRecordingOff
	case !auto && mode != CallRecordingOptional:
		return ErrRecordingNotOpt
	}
	l, err := s.load(tenantID, callID)
	if err != nil {
		return ErrNotFound
	}
	if l.EndedAt != nil || l.HandledByUserID == nil || *l.HandledByUserID != userID {
		return ErrCallNotInProgress
	}
	if _, err := s.rec.Start(tenantID, callID); err != nil {
		return err
	}
	s.fresh().Model(&models.CallLog{}).Where(`"tenantId" = ? AND "callId" = ?`, tenantID, callID).
		Update("recordingStatus", recording.StatusRecording)
	s.bcast.EmitToRoom("/", UserRoom(tenantID, userID), "call.recording", map[string]interface{}{"callId": callID, "recording": true})
	return nil
}

// StopRecording encerra a gravação em curso (modo opcional) e a envia ao armazenamento.
func (s *Service) StopRecording(ctx context.Context, tenantID uuid.UUID, userID int, callID string) error {
	l, err := s.load(tenantID, callID)
	if err != nil {
		return ErrNotFound
	}
	if l.HandledByUserID == nil || *l.HandledByUserID != userID {
		return ErrCallNotInProgress
	}
	if s.rec == nil || !s.rec.Active(tenantID, callID) {
		return ErrRecordingNone
	}
	s.finalizeRecording(ctx, tenantID, callID)
	s.bcast.EmitToRoom("/", UserRoom(tenantID, userID), "call.recording", map[string]interface{}{"callId": callID, "recording": false})
	return nil
}
