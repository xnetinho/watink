package calls

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/recording"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	upErr   error
	deleted []string
}

func newMemStore() *memStore { return &memStore{objects: map[string][]byte{}} }

func (m *memStore) Upload(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	if m.upErr != nil {
		return m.upErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.objects[key] = b
	m.mu.Unlock()
	return nil
}
func (m *memStore) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (m *memStore) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://s3.local/" + key + "?sig=1", nil
}
func (m *memStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, key)
	delete(m.objects, key)
	return nil
}
func (m *memStore) Describe() map[string]any { return nil }
func (m *memStore) has(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[key]
	return ok
}

func (r *rig) withRecording(t *testing.T, store *memStore) {
	t.Helper()
	if store == nil {
		r.svc.WithRecording(NewRecording(nil, t.TempDir()))
		return
	}
	r.svc.WithRecording(NewRecording(store, t.TempDir()))
}

func (r *rig) setMode(t *testing.T, mode string) {
	t.Helper()
	require.NoError(t, r.db.Exec(`INSERT INTO "Settings" (key, value, "tenantId") VALUES (?, ?, ?)
		ON CONFLICT (key, "tenantId") DO UPDATE SET value = EXCLUDED.value`, SettingRecordingMode, mode, r.tenant).Error)
}

func pcmFrame(amp float64, hz float64, idx int) []byte {
	b := make([]byte, recording.FrameSamples*2)
	for i := 0; i < recording.FrameSamples; i++ {
		n := idx*recording.FrameSamples + i
		binary.LittleEndian.PutUint16(b[i*2:], uint16(int16(amp*math.Sin(2*math.Pi*hz*float64(n)/recording.SampleRate))))
	}
	return b
}

func TestRecordingMode_AbsentAndInvalidAreOff(t *testing.T) {
	r := newRig(t)
	assert.Equal(t, CallRecordingOff, r.svc.recordingMode(r.tenant), "empresa que nunca configurou não grava")
	r.setMode(t, "auto")
	assert.Equal(t, CallRecordingAuto, r.svc.recordingMode(r.tenant))
	assert.Equal(t, CallRecordingOff, r.svc.recordingMode(uuid.New()), "o modo de outra empresa não vaza")
	r.setMode(t, "valor-corrompido")
	assert.Equal(t, CallRecordingOff, r.svc.recordingMode(r.tenant), "valor inválido nunca liga a gravação")
}

// 7.11: os três modos.
func TestStartRecording_OffRejects(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	uid := answeredCall(t, r, "REC-1")
	err := r.svc.StartRecording(r.tenant, uid, "REC-1", false)
	assert.Equal(t, ErrRecordingOff, err, "modo off (padrão) rejeita")
	assert.False(t, r.svc.Recording().Active(r.tenant, "REC-1"))
}

func TestStartRecording_OptionalAllowsManualOnly(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-2")

	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-2", false))
	assert.True(t, r.svc.Recording().Active(r.tenant, "REC-2"))
	assert.Equal(t, recording.StatusRecording, r.log(t, "REC-2").RecordingStatus)
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, uid), "call.recording"), "o operador vê o indicador de gravando")
	assert.Equal(t, ErrRecordingActive, r.svc.StartRecording(r.tenant, uid, "REC-2", false), "não grava duas vezes")
}

func TestStartRecording_AutoModeRejectsManualButAllowsAutomatic(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "auto")
	uid := answeredCall(t, r, "REC-3")
	assert.Equal(t, ErrRecordingNotOpt, r.svc.StartRecording(r.tenant, uid, "REC-3", false), "no modo auto o operador não liga/desliga")
	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-3", true))
}

func TestStartRecording_WithoutS3IsRejected(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, nil)
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-4")
	assert.Equal(t, ErrRecordingNoS3, r.svc.StartRecording(r.tenant, uid, "REC-4", false), "instalação sem S3 rejeita")
	assert.False(t, r.svc.Recording().Available())
}

func TestStartRecording_OnlyTheHandlingOperatorAndOnlyInProgress(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-5")
	other := r.users["setor_fila_A"].ID
	assert.Equal(t, ErrCallNotInProgress, r.svc.StartRecording(r.tenant, other, "REC-5", false), "outro operador não grava a chamada")
	assert.Equal(t, ErrNotFound, r.svc.StartRecording(r.tenant, uid, "NAO-EXISTE", false))
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("REC-5", "user_ended", 3)))
	assert.Equal(t, ErrCallNotInProgress, r.svc.StartRecording(r.tenant, uid, "REC-5", false), "chamada encerrada não grava")
}

// Fluxo completo: grava, encerra a chamada, o MP3 vai ao S3 e o registro aponta a chave.
func TestRecording_EndOfCallUploadsAndRecordsKey(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-6")
	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-6", false))

	for i := 0; i < 100; i++ { // 2 s de áudio dos dois lados
		r.svc.Recording().Feed(r.tenant, "REC-6", true, pcmFrame(8000, 440, i))
		r.svc.Recording().Feed(r.tenant, "REC-6", false, pcmFrame(6000, 880, i))
	}
	time.Sleep(2200 * time.Millisecond)
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("REC-6", "user_ended", 2)))

	l := r.log(t, "REC-6")
	assert.Equal(t, recording.StatusReady, l.RecordingStatus)
	assert.Equal(t, recording.ObjectKey(r.tenant, "REC-6"), l.RecordingKey, "o banco guarda só a chave")
	assert.NotContains(t, l.RecordingKey, "http", "nunca uma URL assinada")
	assert.True(t, store.has(l.RecordingKey), "o MP3 está no armazenamento")
	assert.GreaterOrEqual(t, l.RecordingDurationSec, 1)
	assert.False(t, r.svc.Recording().Active(r.tenant, "REC-6"), "o gravador é liberado")
}

func TestRecording_UploadFailureMarksFailedAndExposesNothing(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	store.upErr = assertErr("s3 fora do ar")
	r.withRecording(t, store)
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-7")
	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-7", false))
	r.svc.Recording().Feed(r.tenant, "REC-7", true, pcmFrame(8000, 440, 0))
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("REC-7", "user_ended", 1)))
	l := r.log(t, "REC-7")
	assert.Equal(t, recording.StatusFailed, l.RecordingStatus, "o histórico indica que a gravação falhou")
	assert.Empty(t, l.RecordingKey, "sem chave: nenhum áudio parcial exposto")
	assert.Equal(t, StatusEnded, l.Status, "a chamada em si é registrada normalmente")
	assert.Empty(t, store.objects)
}

func TestRecording_InterruptedCallDiscardsRecording(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-8")
	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-8", false))
	require.True(t, r.svc.Recording().Active(r.tenant, "REC-8"))

	raw := []byte(`{"sessionId":"` + itoa(r.waA.ID) + `"}`)
	require.NoError(t, r.svc.HandleReset(ctx, r.tenant, raw))
	assert.False(t, r.svc.Recording().Active(r.tenant, "REC-8"), "o gravador de uma chamada interrompida é descartado")
	assert.Empty(t, store.objects)
}

func TestStopRecording_Manual(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)
	r.setMode(t, "optional")
	uid := answeredCall(t, r, "REC-9")
	require.NoError(t, r.svc.StartRecording(r.tenant, uid, "REC-9", false))
	r.svc.Recording().Feed(r.tenant, "REC-9", true, pcmFrame(8000, 440, 0))
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, ErrCallNotInProgress, r.svc.StopRecording(ctx, r.tenant, r.users["setor_fila_A"].ID, "REC-9"))
	require.NoError(t, r.svc.StopRecording(ctx, r.tenant, uid, "REC-9"))
	assert.Equal(t, recording.StatusReady, r.log(t, "REC-9").RecordingStatus)
	assert.Equal(t, ErrRecordingNone, r.svc.StopRecording(ctx, r.tenant, uid, "REC-9"), "já parou")
}

func TestRecordingKeys_MatchTheSettingsController(t *testing.T) {
	assert.Equal(t, "callRecordingMode", SettingRecordingMode)
	assert.Equal(t, "callRecordingAckBy", SettingRecordingAckBy)
	assert.Equal(t, "callRecordingAckAt", SettingRecordingAckAt)
	for in, want := range map[string]string{"": "off", "off": "off", "AUTO": "auto", " optional ": "optional", "x": "off"} {
		assert.Equal(t, want, NormalizeRecordingMode(in), "entrada %q", in)
	}
}

var _ = models.CallLog{}

// 7.11: no modo auto a gravação começa ao CONECTAR o áudio, sem o operador pedir;
// no modo off, conectar o áudio não grava nada.
func TestServeAudio_AutoModeStartsRecordingOnConnect(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)
	r.setMode(t, "auto")
	uid := answeredCall(t, r, "AUTO-1")
	eng := newFakeEngine(t)

	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-1")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn

	require.Eventually(t, func() bool { return r.svc.Recording().Active(r.tenant, "AUTO-1") }, 2*time.Second, 10*time.Millisecond,
		"o modo auto grava assim que o áudio conecta")
	assert.Equal(t, recording.StatusRecording, r.log(t, "AUTO-1").RecordingStatus)
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, uid), "call.recording"), "o operador vê o indicador de gravação")
}

// Bug real: na chamada de SAÍDA o navegador abre o áudio assim que disca, ainda tocando. O modo
// automático gravava o toque e, se o contato recusasse ou não atendesse, sobrava uma "gravação"
// de uma chamada que nunca aconteceu. A gravação só começa quando a chamada é atendida.
func TestServeAudio_AutoMode_OutgoingStillRingingDoesNotRecord(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "auto")
	placeOutgoing(t, r, "AUTO-RING")
	uid := r.users["da_fila_A"].ID
	eng := newFakeEngine(t)

	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-RING")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn
	time.Sleep(300 * time.Millisecond)

	assert.False(t, r.svc.Recording().Active(r.tenant, "AUTO-RING"), "tocando, ninguém atendeu: nada a gravar")
	assert.Empty(t, r.log(t, "AUTO-RING").RecordingStatus)
}

// ...e quando o contato atende, a gravação começa nesse instante, sem o operador pedir.
func TestServeAudio_AutoMode_OutgoingStartsRecordingWhenAnswered(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "auto")
	placeOutgoing(t, r, "AUTO-ANS")
	uid := r.users["da_fila_A"].ID
	eng := newFakeEngine(t)

	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-ANS")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn
	require.False(t, r.svc.Recording().Active(r.tenant, "AUTO-ANS"))

	require.NoError(t, r.svc.HandleState(ctx, r.tenant, stateEvent("AUTO-ANS", "active")))
	require.Eventually(t, func() bool { return r.svc.Recording().Active(r.tenant, "AUTO-ANS") }, 2*time.Second, 10*time.Millisecond,
		"atendida, a gravação automática começa")
	assert.Equal(t, recording.StatusRecording, r.log(t, "AUTO-ANS").RecordingStatus)
}

// Recusada ou sem resposta: o registro fica sem gravação nenhuma.
func TestServeAudio_AutoMode_OutgoingNeverAnsweredLeavesNoRecording(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)
	r.setMode(t, "auto")
	placeOutgoing(t, r, "AUTO-NONE")
	uid := r.users["da_fila_A"].ID
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-NONE")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn

	raw, _ := json.Marshal(map[string]interface{}{"callId": "AUTO-NONE", "endReason": "declined", "durationSecs": 0, "direction": "outgoing"})
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, raw))

	l := r.log(t, "AUTO-NONE")
	assert.Empty(t, l.RecordingStatus, "chamada recusada não gera gravação")
	assert.Empty(t, l.RecordingKey)
	store.mu.Lock()
	assert.Empty(t, store.objects, "nada foi enviado ao armazenamento")
	store.mu.Unlock()
}

func TestServeAudio_OffModeConnectingAudioDoesNotRecord(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	uid := answeredCall(t, r, "AUTO-2")
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-2")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn
	time.Sleep(300 * time.Millisecond)
	assert.False(t, r.svc.Recording().Active(r.tenant, "AUTO-2"), "modo off (padrão) nunca grava")
	assert.Empty(t, r.log(t, "AUTO-2").RecordingStatus)
}

func TestServeAudio_AutoModeWithoutS3RecordsNothingAndCallContinues(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, nil)
	r.setMode(t, "auto")
	uid := answeredCall(t, r, "AUTO-3")
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "AUTO-3")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn
	time.Sleep(300 * time.Millisecond)
	assert.False(t, r.svc.Recording().Active(r.tenant, "AUTO-3"), "sem S3 não grava")
	assert.Nil(t, r.log(t, "AUTO-3").EndedAt, "a chamada NÃO é encerrada por não poder gravar")
	assert.Empty(t, r.pub.cmds("call.end"), "e o engine não é mandado desligar")
}

// Gravações não expiram: nenhum caminho de código remove uma gravação por idade. Só a
// exclusão manual (DeleteRecording) a apaga. Este teste cria uma gravação "muito antiga" e
// garante que nada do ciclo normal (encerrar, reset, novas chamadas) a toca.
func TestRecording_NeverExpiresByAge(t *testing.T) {
	r := newRig(t)
	store := newMemStore()
	r.withRecording(t, store)

	old := time.Now().AddDate(-5, 0, 0)
	key := recording.ObjectKey(r.tenant, "OLD-1")
	store.objects[key] = []byte("mp3 de 5 anos atrás")
	require.NoError(t, r.db.Create(&models.CallLog{
		TenantID: r.tenant, CallID: "OLD-1", WhatsappID: r.waA.ID, Direction: "incoming", Status: StatusEnded,
		StartedAt: old, RecordingKey: key, RecordingStatus: recording.StatusReady,
	}).Error)

	// o resto do sistema segue funcionando em volta: novas chamadas, encerramentos, reset
	ringing(t, r, "NEW-1")
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("NEW-1", "timeout", 0)))
	raw := []byte(`{"sessionId":"` + itoa(r.waA.ID) + `"}`)
	require.NoError(t, r.svc.HandleReset(ctx, r.tenant, raw))

	l := r.log(t, "OLD-1")
	assert.Equal(t, recording.StatusReady, l.RecordingStatus, "a gravação antiga continua pronta")
	assert.Equal(t, key, l.RecordingKey)
	assert.True(t, store.has(key), "o arquivo continua no armazenamento")
	assert.Empty(t, store.deleted, "nada foi apagado automaticamente")
}
