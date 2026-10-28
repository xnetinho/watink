package whatsapp

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type fakePresence struct {
	mu    sync.Mutex
	sent  []types.Presence
	fails bool
}

func (f *fakePresence) SendPresence(_ context.Context, s types.Presence) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fails {
		return context.DeadlineExceeded
	}
	f.sent = append(f.sent, s)
	return nil
}
func (f *fakePresence) last() types.Presence {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		return ""
	}
	return f.sent[len(f.sent)-1]
}
func (f *fakePresence) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.sent) }

func newState() *presenceState { return &presenceState{policy: policyNever, idle: idleDuration(0)} }

// never: nunca marca disponível, mesmo com operador ativo.
func TestPresence_NeverNuncaMarcaDisponivel(t *testing.T) {
	st, c := newState(), &fakePresence{}
	if err := st.applyPolicy(c, policyNever, 5); err != nil {
		t.Fatal(err)
	}
	if st.shouldMarkAvailable() {
		t.Fatal("never não marca disponível")
	}
	if ok, _ := st.operatorActive(c, func() {}); ok {
		t.Fatal("operador ativo não muda o never")
	}
	if c.count() != 0 {
		t.Fatalf("nenhum envio esperado, houve %d", c.count())
	}
}

// always: disponível ao aplicar e REAPLICADO a cada conexão (o servidor esquece a presença ao cair).
func TestPresence_AlwaysReaplicaAoConectar(t *testing.T) {
	st, c := newState(), &fakePresence{}
	if err := st.applyPolicy(c, policyAlways, 5); err != nil || c.last() != types.PresenceAvailable {
		t.Fatalf("always deve marcar disponível: %v %q", err, c.last())
	}
	before := c.count()
	if err := st.onConnect(c); err != nil {
		t.Fatal(err)
	}
	if c.count() != before+1 || c.last() != types.PresenceAvailable {
		t.Fatalf("ao reconectar deve reenviar disponível (antes %d, agora %d)", before, c.count())
	}
}

// on_demand: começa indisponível, fica disponível com operador e volta a indisponível depois do tempo ocioso.
func TestPresence_OnDemandVoltaAIndisponivelDepoisDoTempo(t *testing.T) {
	st, c := newState(), &fakePresence{}
	if err := st.applyPolicy(c, policyOnDemand, 5); err != nil {
		t.Fatal(err)
	}
	st.idle = 40 * time.Millisecond // só o teste: o mínimo real é 1 minuto
	idle := make(chan struct{})
	if ok, err := st.operatorActive(c, func() { _ = st.goIdle(c); close(idle) }); !ok || err != nil {
		t.Fatalf("operador deve deixar disponível: %v %v", ok, err)
	}
	if c.last() != types.PresenceAvailable {
		t.Fatalf("com operador: %q", c.last())
	}
	select {
	case <-idle:
	case <-time.After(2 * time.Second):
		t.Fatal("o temporizador não disparou")
	}
	if c.last() != types.PresenceUnavailable {
		t.Fatalf("sem operador deve voltar a indisponível, está %q", c.last())
	}
}

// Atividade do operador renova o prazo: não pode voltar a indisponível enquanto ele ainda está olhando.
func TestPresence_OnDemandAtividadeRenovaOPrazo(t *testing.T) {
	st, c := newState(), &fakePresence{}
	_ = st.applyPolicy(c, policyOnDemand, 5)
	st.idle = 120 * time.Millisecond
	fired := make(chan struct{}, 4)
	cb := func() { _ = st.goIdle(c); fired <- struct{}{} }
	_, _ = st.operatorActive(c, cb)
	time.Sleep(70 * time.Millisecond)
	_, _ = st.operatorActive(c, cb) // renova
	time.Sleep(70 * time.Millisecond)
	select {
	case <-fired:
		t.Fatal("voltou a indisponível mesmo com o operador renovando o prazo")
	default:
	}
	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("deveria voltar a indisponível após parar de renovar")
	}
}

// Trocar a política desarma o temporizador e acerta o estado: on_demand -> never volta a indisponível na hora.
func TestPresence_TrocarPoliticaDesarmaOTemporizador(t *testing.T) {
	st, c := newState(), &fakePresence{}
	_ = st.applyPolicy(c, policyOnDemand, 5)
	_, _ = st.operatorActive(c, func() {})
	if st.timer == nil {
		t.Fatal("temporizador deveria estar armado")
	}
	_ = st.applyPolicy(c, policyNever, 5)
	if st.timer != nil || c.last() != types.PresenceUnavailable {
		t.Fatalf("never deve desarmar e voltar a indisponível (timer=%v, último=%q)", st.timer, c.last())
	}
}

// Não repete envio quando o estado não mudou (cada envio é uma operação no WhatsApp).
func TestPresence_NaoReenviaSeNaoMudou(t *testing.T) {
	st, c := newState(), &fakePresence{}
	_ = st.applyPolicy(c, policyAlways, 5)
	_ = st.applyPolicy(c, policyAlways, 5)
	_, _ = st.operatorActive(c, func() {})
	if c.count() != 1 {
		t.Fatalf("esperava 1 envio, houve %d", c.count())
	}
}

// Falha do WhatsApp ao enviar não pode marcar o estado como aplicado (senão nunca tentaria de novo).
func TestPresence_FalhaNaoMarcaComoAplicado(t *testing.T) {
	st, c := newState(), &fakePresence{fails: true}
	if err := st.applyPolicy(c, policyAlways, 5); err == nil {
		t.Fatal("o erro deve ser devolvido")
	}
	c.fails = false
	if err := st.onConnect(c); err != nil || c.last() != types.PresenceAvailable {
		t.Fatalf("depois de falhar, a próxima tentativa precisa enviar: %v %q", err, c.last())
	}
}

func TestPresence_ValorInvalidoViraNever(t *testing.T) {
	for _, bad := range []string{"", "ALWAYS", "xyz", "on-demand"} {
		if got := normalizePolicy(bad); got != policyNever {
			t.Errorf("%q -> %q, esperado never (o mais seguro: não silencia o celular)", bad, got)
		}
	}
	if idleDuration(0) != 5*time.Minute || idleDuration(-1) != 5*time.Minute || idleDuration(30) != 30*time.Minute {
		t.Fatal("tempo ocioso: padrão 5 min, valor válido respeitado")
	}
}
