package calls

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ringing cria uma chamada recebida tocando para dois operadores elegíveis.
func ringing(t *testing.T, r *rig, callID string) {
	t.Helper()
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "setor_fila_A", "receive")
	r.online("da_fila_A", "setor_fila_A")
	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, incoming(callID, r.waA.ID, "5511999990010@s.whatsapp.net", "5511999990010")))
}

// 6.5: dois atendimentos concorrentes → só um vence, o outro recebe "já atendida".
func TestAccept_ConcurrentOnlyOneWins(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "ACC-1")
	a, b := r.users["da_fila_A"].ID, r.users["setor_fila_A"].ID

	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i, uid := range []int{a, b} {
		wg.Add(1)
		go func(i, uid int) {
			defer wg.Done()
			<-start
			_, errs[i] = r.svc.Accept(ctx, r.tenant, uid, "ACC-1")
		}(i, uid)
	}
	close(start)
	wg.Wait()

	wins, lost := 0, 0
	for _, e := range errs {
		switch e {
		case nil:
			wins++
		case ErrAlreadyAnswered:
			lost++
		default:
			t.Fatalf("erro inesperado: %v", e)
		}
	}
	assert.Equal(t, 1, wins, "exatamente um atendimento vence")
	assert.Equal(t, 1, lost, "o outro recebe 'já atendida'")
	assert.Len(t, r.pub.cmds("call.accept"), 1, "o engine é mandado atender uma única vez")

	l := r.log(t, "ACC-1")
	require.NotNil(t, l.HandledByUserID)
	assert.Contains(t, []int{a, b}, *l.HandledByUserID)
	assert.NotNil(t, l.AnsweredAt)
}

func TestAccept_UnknownCallAndOtherTenant(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "ACC-2")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "NAO-EXISTE")
	assert.Equal(t, ErrNotFound, err)
	_, err = r.svc.Accept(ctx, r.other, r.users["de_outra_empresa"].ID, "ACC-2")
	assert.Equal(t, ErrNotFound, err, "outra empresa nunca enxerga a chamada")
	assert.Empty(t, r.pub.cmds("call.accept"))
}

func TestAccept_UserAlreadyInCallIsDenied(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "ACC-3")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "ACC-3")
	require.NoError(t, err)

	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, incoming("ACC-4", r.waA.ID, "5511999990011@s.whatsapp.net", "5511999990011")))
	_, err = r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "ACC-4")
	assert.Equal(t, ErrUserBusy, err, "quem já está em chamada não atende outra")
	assert.Equal(t, "ACC-3", func() string { return r.log(t, "ACC-3").CallID }(), "a chamada atual segue intacta")
	assert.Nil(t, r.log(t, "ACC-4").HandledByUserID)
}

func TestAccept_EngineFailureRollsAssignmentBack(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "ACC-5")
	r.pub.err = assertErr("amqp fora")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "ACC-5")
	assert.Error(t, err)
	assert.Nil(t, r.log(t, "ACC-5").HandledByUserID, "sem confirmar ao engine, a chamada volta a ficar disponível")
	r.pub.err = nil
	_, err = r.svc.Accept(ctx, r.tenant, r.users["setor_fila_A"].ID, "ACC-5")
	assert.NoError(t, err, "outro operador ainda consegue atender")
}

func TestReject_EndsAndOnlyOnceAndCommandsEngine(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "REJ-1")
	require.NoError(t, r.svc.Reject(ctx, r.tenant, r.users["da_fila_A"].ID, "REJ-1"))

	l := r.log(t, "REJ-1")
	assert.Equal(t, StatusRejected, l.Status)
	assert.Equal(t, "declined", l.EndReason)
	assert.Len(t, r.pub.cmds("call.reject"), 1, "o reject só sai por comando do operador")
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"), "o toque some para todos")

	assert.Equal(t, ErrAlreadyAnswered, r.svc.Reject(ctx, r.tenant, r.users["setor_fila_A"].ID, "REJ-1"))
	assert.Len(t, r.pub.cmds("call.reject"), 1)
}

func TestReject_AfterAnsweredIsDenied(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "REJ-2")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "REJ-2")
	require.NoError(t, err)
	assert.Equal(t, ErrAlreadyAnswered, r.svc.Reject(ctx, r.tenant, r.users["setor_fila_A"].ID, "REJ-2"))
	assert.Empty(t, r.pub.cmds("call.reject"))
}

func TestEnd_OnlyTheHandlingOperator(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "END-1")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "END-1")
	require.NoError(t, err)

	assert.Equal(t, ErrNotActive, r.svc.End(ctx, r.tenant, r.users["setor_fila_A"].ID, "END-1"), "outro operador não encerra")
	assert.Empty(t, r.pub.cmds("call.end"))
	require.NoError(t, r.svc.End(ctx, r.tenant, r.users["da_fila_A"].ID, "END-1"))
	assert.Len(t, r.pub.cmds("call.end"), 1)
}

func ended(callID, reason string, secs int) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{"callId": callID, "endReason": reason, "durationSecs": secs, "direction": "incoming"})
	return b
}

// 6.6: ao encerrar, duração real + resumo de qualidade + mensagem de sistema no ticket.
func TestHandleEnded_FinalizesWithDurationQualityAndMessage(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "FIN-1")
	_, err := r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "FIN-1")
	require.NoError(t, err)

	rtt := 100.0
	for i := 0; i < 3; i++ {
		raw, _ := json.Marshal(map[string]interface{}{"callId": "FIN-1", "rttMs": rtt, "lossPct": float64(i), "jitterMs": 10.0})
		require.NoError(t, r.svc.HandleQuality(ctx, r.tenant, raw))
	}
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("FIN-1", "user_ended", 73)))

	l := r.log(t, "FIN-1")
	assert.Equal(t, StatusEnded, l.Status)
	assert.Equal(t, 73, l.DurationSec)
	assert.NotNil(t, l.EndedAt)
	require.NotNil(t, l.LossMax)
	assert.InDelta(t, 2.0, *l.LossMax, 0.05)
	assert.InDelta(t, 1.0, *l.LossAvg, 0.05)
	assert.InDelta(t, 100.0, *l.RttAvg, 0.05)
	require.NotNil(t, l.MosEstimated)
	assert.Equal(t, 3, l.QualitySamples)

	var msgs []models.Message
	require.NoError(t, r.db.Where(`"ticketId" = ? AND "mediaType" = 'call'`, *l.TicketID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Chamada de voz recebida", msgs[0].Body)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(msgs[0].DataJson), &data))
	assert.EqualValues(t, 73, data["durationSec"])
	assert.Equal(t, "FIN-1", data["callId"])
	assert.Equal(t, 1, r.bc.to("chat:"+itoa(*l.TicketID), "appMessage"))
}

func TestHandleEnded_IsIdempotent(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "FIN-2")
	_, _ = r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "FIN-2")
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("FIN-2", "user_ended", 10)))
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("FIN-2", "user_ended", 99)))

	assert.Equal(t, 10, r.log(t, "FIN-2").DurationSec, "a segunda entrega não sobrescreve")
	var n int64
	r.db.Model(&models.Message{}).Where(`id = ?`, callMessageID("FIN-2")).Count(&n)
	assert.EqualValues(t, 1, n, "uma única mensagem no ticket")
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"))
}

func TestHandleEnded_UnansweredBecomesMissed(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "FIN-3")
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("FIN-3", "timeout", 0)))
	assert.Equal(t, StatusMissed, r.log(t, "FIN-3").Status, "ninguém atendeu → perdida")
}

func TestHandleEnded_FailureAndUnknownCall(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "FIN-4")
	_, _ = r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "FIN-4")
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("FIN-4", "failed", 5)))
	assert.Equal(t, StatusFailed, r.log(t, "FIN-4").Status)
	assert.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("NAO-EXISTE", "user_ended", 1)), "chamada desconhecida não é erro")
}

// 6.8: engine reiniciou a sessão → toda chamada aberta daquela conexão vira interrompida.
func TestHandleReset_InterruptsOpenCallsOfThatConnectionOnly(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "RST-1")
	r.grant(t, "da_fila_B", "receive")
	r.online("da_fila_B")
	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, incoming("RST-OTHER", r.waB.ID, "5511999990012@s.whatsapp.net", "5511999990012")))
	require.Equal(t, StatusRinging, r.log(t, "RST-OTHER").Status, "pré-condição: a outra conexão também está tocando")

	raw, _ := json.Marshal(map[string]interface{}{"sessionId": itoa(r.waA.ID)})
	require.NoError(t, r.svc.HandleReset(ctx, r.tenant, raw))

	l := r.log(t, "RST-1")
	assert.Equal(t, StatusInterrupted, l.Status)
	assert.Equal(t, "interrupted", l.EndReason)
	assert.NotNil(t, l.EndedAt)
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"), "o toque some dos operadores")
	assert.NotEqual(t, StatusInterrupted, r.log(t, "RST-OTHER").Status, "outra conexão não é afetada")

	require.NoError(t, r.svc.HandleReset(ctx, r.tenant, raw))
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"), "reset repetido não reemite")
}

func TestHandleEnded_InterruptedReasonMarksInterrupted(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "RST-2")
	_, _ = r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "RST-2")
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("RST-2", "interrupted", 4)))
	assert.Equal(t, StatusInterrupted, r.log(t, "RST-2").Status)
}

// Telemetria: só o operador que assumiu a chamada a recebe.
func TestHandleQuality_DeliveredOnlyToTheHandlingOperator(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "Q-1")
	raw := func(loss float64) json.RawMessage {
		b, _ := json.Marshal(map[string]interface{}{"callId": "Q-1", "rttMs": 50.0, "lossPct": loss, "jitterMs": 5.0})
		return b
	}
	require.NoError(t, r.svc.HandleQuality(ctx, r.tenant, raw(0)))
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["da_fila_A"].ID), "call.quality"), "antes de alguém assumir não se entrega")

	_, _ = r.svc.Accept(ctx, r.tenant, r.users["da_fila_A"].ID, "Q-1")
	require.NoError(t, r.svc.HandleQuality(ctx, r.tenant, raw(7)))
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, r.users["da_fila_A"].ID), "call.quality"))
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["setor_fila_A"].ID), "call.quality"), "outro operador não recebe")
	assert.Zero(t, r.bc.to("tenant:"+r.tenant.String(), "call.quality"), "e a empresa toda também não")

	var got map[string]interface{}
	r.bc.mu.Lock()
	for _, e := range r.bc.evs {
		if e.event == "call.quality" {
			got = e.payload.(map[string]interface{})
		}
	}
	r.bc.mu.Unlock()
	assert.Equal(t, LevelPoor, got["level"], "perda de 7% é nível ruim")
	assert.Contains(t, got["alerts"], AlertLoss)
	assert.NotNil(t, got["mosEstimated"])
}
