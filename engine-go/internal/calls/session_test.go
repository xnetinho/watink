package calls

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (t *fakeTimer) Stop() bool { was := !t.stopped; t.stopped = true; return was }

type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

func (c *fakeClock) AfterFunc(d time.Duration, f func()) stopper {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{d: d, f: f}
	c.timers = append(c.timers, t)
	return t
}

// fire dispara os temporizadores vivos com a duração d.
func (c *fakeClock) fire(d time.Duration) int {
	c.mu.Lock()
	var due []*fakeTimer
	for _, t := range c.timers {
		if t.d == d && !t.stopped {
			t.stopped = true
			due = append(due, t)
		}
	}
	c.mu.Unlock()
	for _, t := range due {
		t.f()
	}
	return len(due)
}

type fakeHandle struct {
	mu      sync.Mutex
	hooks   Hooks
	log     []string
	panicOn string
	started chan struct{}

	media   MediaHooks
	fed     []float32
	rtt     int
	relayUp bool
}

func (h *fakeHandle) rec(s string) {
	h.mu.Lock()
	h.log = append(h.log, s)
	h.mu.Unlock()
	if h.panicOn == s {
		panic("boom " + s)
	}
}
func (h *fakeHandle) calls() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.log...)
}
func (h *fakeHandle) has(s string) bool {
	for _, c := range h.calls() {
		if c == s {
			return true
		}
	}
	return false
}
func (h *fakeHandle) SetHooks(k Hooks) { h.hooks = k }
func (h *fakeHandle) HandleOffer(context.Context, *waBinary.Node, types.JID) {
	h.rec("offer")
	if h.started != nil {
		<-h.started
	}
}
func (h *fakeHandle) HandleAccept(context.Context, *waBinary.Node, types.JID)    { h.rec("accept-evt") }
func (h *fakeHandle) HandleTransport(context.Context, *waBinary.Node, types.JID) { h.rec("transport") }
func (h *fakeHandle) HandleTerminate(*waBinary.Node) {
	h.rec("terminate-evt")
	h.hooks.OnEnded(State{EndReason: "user_ended"})
}
func (h *fakeHandle) HandleRelayLatency(*waBinary.Node)   { h.rec("latency") }
func (h *fakeHandle) SendPreaccept(context.Context) error { h.rec("preaccept"); return nil }
func (h *fakeHandle) Accept(context.Context, string) error {
	h.rec("accept")
	// O CallManager real, ao atender, vai a "connecting" e emite o estado (callmanager.go:
	// TransitionLocalAccepted + emitState). O fake reproduz esse contrato.
	if h.hooks.OnState != nil {
		h.hooks.OnState(State{State: "connecting", Direction: "incoming"})
	}
	return nil
}
func (h *fakeHandle) Reject(context.Context, string) error {
	h.rec("reject")
	h.hooks.OnEnded(State{EndReason: "declined"})
	return nil
}
func (h *fakeHandle) End(_ context.Context, reason string) error {
	h.rec("end:" + reason)
	h.hooks.OnEnded(State{EndReason: reason, DurationSecs: 7})
	return nil
}
func (h *fakeHandle) Start(context.Context, string, types.JID) error { h.rec("start"); return nil }
func (h *fakeHandle) SetMedia(k MediaHooks)                          { h.media = k }
func (h *fakeHandle) FeedPCM(p []float32) {
	h.mu.Lock()
	h.fed = append(h.fed, p...)
	h.mu.Unlock()
}
func (h *fakeHandle) RelayRTTMs() (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rtt, h.rtt > 0
}
func (h *fakeHandle) RelayConnected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.relayUp
}
func (h *fakeHandle) Abandon(reason string) {
	h.rec("abandon:" + reason)
	h.hooks.OnEnded(State{EndReason: reason})
}

type published struct {
	typ string
	p   map[string]interface{}
}

type rig struct {
	t       *testing.T
	s       *Session
	clk     *fakeClock
	mu      sync.Mutex
	events  []published
	handles []*fakeHandle
}

func newRig(t *testing.T, proxied bool, lookup func(context.Context, types.JID) (types.JID, error)) *rig {
	r := &rig{t: t, clk: &fakeClock{}}
	r.s = NewSession(SessionConfig{
		ID: 7, TenantID: "tenant-1", Proxied: proxied, LookupPN: lookup, Clock: r.clk,
		Publish: func(_ string, _ int, typ string, p map[string]interface{}) {
			r.mu.Lock()
			r.events = append(r.events, published{typ, p})
			r.mu.Unlock()
		},
		NewHandle: func(core.VoipSocket) Handle {
			h := &fakeHandle{}
			r.mu.Lock()
			r.handles = append(r.handles, h)
			r.mu.Unlock()
			return h
		},
	})
	return r
}

func (r *rig) evs(typ string) []map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []map[string]interface{}
	for _, e := range r.events {
		if e.typ == typ {
			out = append(out, e.p)
		}
	}
	return out
}

func (r *rig) handle(i int) *fakeHandle {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i >= len(r.handles) {
		return nil
	}
	return r.handles[i]
}

func (r *rig) nHandles() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.handles) }

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condição não atendida: %s", what)
}

const callA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
const callB = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

func pn(user string) types.JID  { return types.NewJID(user, types.DefaultUserServer) }
func lid(user string) types.JID { return types.NewJID(user, types.HiddenUserServer) }

func offer(id string, from types.JID) *events.CallOffer {
	return &events.CallOffer{BasicCallMeta: types.BasicCallMeta{From: from, CallCreator: from, CallID: id}, Data: &waBinary.Node{Tag: "offer"}}
}

func TestOffer_PublishesIncomingWithoutPreaccept(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))

	in := r.evs("call.incoming")
	if len(in) != 1 || in[0]["callId"] != callA || in[0]["callerPn"] != "5511999990001" ||
		in[0]["direction"] != "incoming" || in[0]["peer"] != "5511999990001@s.whatsapp.net" || in[0]["sessionId"] != "7" {
		t.Fatalf("payload call.incoming: %v", in)
	}
	eventually(t, "oferta processada", func() bool { return r.handle(0).has("offer") })
	if r.handle(0).has("preaccept") {
		t.Fatal("preaccept foi enviado antes do call.ready")
	}
}

func TestReady_SendsPreaccept(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	if err := r.s.Ready(context.Background(), callA); err != nil {
		t.Fatal(err)
	}
	eventually(t, "preaccept", func() bool { return r.handle(0).has("preaccept") })
}

func TestReadyTimeout_AbandonsWithoutRejectOrTerminate(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	eventually(t, "oferta processada", func() bool { return r.handle(0).has("offer") })

	if n := r.clk.fire(readyTimeout); n != 1 {
		t.Fatalf("esperava 1 temporizador de %v, havia %d", readyTimeout, n)
	}
	m := r.evs("call.missed")
	if len(m) != 1 || m[0]["reason"] != ReasonNoOperator {
		t.Fatalf("call.missed: %v", m)
	}
	h := r.handle(0)
	if !h.has("abandon:no_operator") || h.has("reject") || h.has("preaccept") {
		t.Fatalf("chamadas ao gerenciador: %v", h.calls())
	}
	for _, c := range h.calls() {
		if len(c) >= 3 && c[:3] == "end" {
			t.Fatalf("nenhum terminate pode ser enviado: %v", h.calls())
		}
	}
	if r.s.get(callA) != nil {
		t.Fatal("o slot da conexão não foi liberado")
	}
}

func TestReadyBeforeTimeout_DoesNotAbandon(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	r.clk.fire(readyTimeout)
	time.Sleep(30 * time.Millisecond)
	if len(r.evs("call.missed")) != 0 || r.handle(0).has("abandon:no_operator") {
		t.Fatal("chamada confirmada não pode ser abandonada")
	}
}

func TestBusy_SecondOfferIsIgnoredNotRejected(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	r.s.OnOffer(context.Background(), offer(callB, pn("5511999990002")))

	m := r.evs("call.missed")
	if len(m) != 1 || m[0]["callId"] != callB || m[0]["reason"] != ReasonBusy {
		t.Fatalf("call.missed: %v", m)
	}
	if r.nHandles() != 1 {
		t.Fatalf("a oferta ocupada não pode criar gerenciador (nada é enviado): %d", r.nHandles())
	}
	if len(r.evs("call.incoming")) != 1 {
		t.Fatal("só a primeira chamada deve tocar")
	}
}

func TestUnsupportedTypes_AreIgnored(t *testing.T) {
	r := newRig(t, false, nil)
	video := offer(callA, pn("5511999990001"))
	video.Data = &waBinary.Node{Tag: "offer", Content: []waBinary.Node{{Tag: "video"}}}
	r.s.OnOffer(context.Background(), video)

	group := offer(callB, pn("5511999990002"))
	group.GroupJID = types.NewJID("123", types.GroupServer)
	r.s.OnOffer(context.Background(), group)

	m := r.evs("call.missed")
	if len(m) != 2 || m[0]["reason"] != ReasonUnsupportedType || m[1]["reason"] != ReasonUnsupportedType {
		t.Fatalf("call.missed: %v", m)
	}
	if r.nHandles() != 0 || len(r.evs("call.incoming")) != 0 {
		t.Fatal("tipo não suportado não pode abrir gerenciador nem tocar")
	}
}

func TestProxy_BlocksOfferAndStart(t *testing.T) {
	r := newRig(t, true, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	m := r.evs("call.missed")
	if len(m) != 1 || m[0]["reason"] != ReasonProxyBlocked {
		t.Fatalf("call.missed: %v", m)
	}
	if r.nHandles() != 0 {
		t.Fatal("proxy: nenhum gerenciador (e portanto nenhum socket de mídia) pode ser criado")
	}
	if err := r.s.Start(context.Background(), callB, "5511999990009@s.whatsapp.net"); err != ErrProxied {
		t.Fatalf("Start com proxy: %v", err)
	}
	if r.nHandles() != 0 {
		t.Fatal("Start com proxy não pode criar gerenciador")
	}
}

func TestCallerPN(t *testing.T) {
	known := func(_ context.Context, l types.JID) (types.JID, error) {
		if l.User == "111" {
			return pn("5511988887777"), nil
		}
		return types.EmptyJID, nil
	}
	cases := []struct {
		name    string
		creator types.JID
		alt     types.JID
		lookup  func(context.Context, types.JID) (types.JID, error)
		want    string
	}{
		{"telefone direto", pn("5511977776666"), types.EmptyJID, nil, "5511977776666"},
		{"lid com alternativo telefone", lid("111"), pn("5511966665555"), nil, "5511966665555"},
		{"lid resolvido pelo mapa", lid("111"), types.EmptyJID, known, "5511988887777"},
		{"lid sem mapa", lid("222"), types.EmptyJID, known, ""},
		{"lid sem store", lid("111"), types.EmptyJID, nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t, false, c.lookup)
			if got := r.s.callerPN(context.Background(), c.creator, c.alt); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestOffer_LidWithoutMapSendsEmptyCallerPn(t *testing.T) {
	r := newRig(t, false, nil)
	ev := offer(callA, lid("222"))
	r.s.OnOffer(context.Background(), ev)
	in := r.evs("call.incoming")
	if len(in) != 1 || in[0]["callerPn"] != "" || in[0]["peer"] != "222@lid" {
		t.Fatalf("payload: %v", in)
	}
}

func TestEvents_RouteToTheRightHandle(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	eventually(t, "accept", func() bool { return r.handle(0).has("accept") })

	meta := types.BasicCallMeta{From: pn("5511999990001"), CallID: callA}
	r.s.OnAccept(context.Background(), &events.CallAccept{BasicCallMeta: meta, Data: &waBinary.Node{Tag: "accept"}})
	r.s.OnTransport(context.Background(), &events.CallTransport{BasicCallMeta: meta, Data: &waBinary.Node{Tag: "transport"}})
	r.s.OnRelayLatency(&events.CallRelayLatency{BasicCallMeta: meta, Data: &waBinary.Node{Tag: "relaylatency"}})
	for _, want := range []string{"accept-evt", "transport", "latency"} {
		w := want
		eventually(t, w, func() bool { return r.handle(0).has(w) })
	}

	other := types.BasicCallMeta{From: pn("5511999990001"), CallID: callB}
	r.s.OnTransport(context.Background(), &events.CallTransport{BasicCallMeta: other, Data: &waBinary.Node{Tag: "transport"}})
	r.s.OnTerminate(&events.CallTerminate{BasicCallMeta: other, Data: &waBinary.Node{Tag: "terminate"}})
	time.Sleep(30 * time.Millisecond)
	if r.handle(0).has("terminate-evt") {
		t.Fatal("evento de outra chamada não pode chegar a esta")
	}
}

func TestTerminate_PublishesEndedAndFreesSlot(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	r.s.OnTerminate(&events.CallTerminate{BasicCallMeta: types.BasicCallMeta{From: pn("5511999990001"), CallID: callA}, Data: &waBinary.Node{Tag: "terminate"}})
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	e := r.evs("call.ended")[0]
	if e["callId"] != callA || e["direction"] != "incoming" || e["endReason"] != "user_ended" || e["callerPn"] != "5511999990001" {
		t.Fatalf("call.ended: %v", e)
	}
	if r.s.get(callA) != nil {
		t.Fatal("slot não liberado")
	}
	r.s.OnOffer(context.Background(), offer(callB, pn("5511999990002")))
	if len(r.evs("call.incoming")) != 2 {
		t.Fatal("depois do término a conexão volta a aceitar chamadas")
	}
}

func TestAcceptedElsewhere_EndsWithoutSendingAnything(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })

	r.s.OnAccept(context.Background(), &events.CallAccept{BasicCallMeta: types.BasicCallMeta{From: pn("5511999990001"), CallID: callA}, Data: &waBinary.Node{Tag: "accept"}})
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	if got := r.evs("call.ended")[0]["endReason"]; got != ReasonAcceptedElse {
		t.Fatalf("endReason=%v", got)
	}
	h := r.handle(0)
	if h.has("reject") || h.has("accept-evt") || h.has("end:user_ended") {
		t.Fatalf("nada pode ser enviado: %v", h.calls())
	}
}

func TestRingTimeout_IncomingAbandonsAfter45s(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })
	if r.clk.fire(ringTimeout) != 1 {
		t.Fatal("o toque de 45 s não foi armado")
	}
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	if r.evs("call.ended")[0]["endReason"] != ReasonTimeout || r.handle(0).has("reject") {
		t.Fatalf("%v %v", r.evs("call.ended"), r.handle(0).calls())
	}
}

func TestAccept_StopsRingTimer(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	if r.clk.fire(ringTimeout) != 0 {
		t.Fatal("atendida, a chamada não pode mais expirar por toque")
	}
}

// Bug real do primeiro teste com duas pessoas: na chamada de SAÍDA o contato atende, a mídia
// conecta, e 45 s depois do início o timer de toque ainda dispara e derruba a chamada ativa
// ("timeout" aos 41 s de conversa). Atendida, ela não pode mais expirar por toque.
func TestStart_AnsweredCallNeverExpiresByRingTimer(t *testing.T) {
	r := newRig(t, false, nil)
	if err := r.s.Start(context.Background(), callA, "5511999990001@s.whatsapp.net"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "start", func() bool { return r.handle(0) != nil && r.handle(0).has("start") })
	r.handle(0).hooks.OnState(State{State: "active", Direction: "outgoing"})
	if r.clk.fire(ringTimeout) != 0 {
		t.Fatal("chamada de saída já ativa: o timer de toque tem de ter sido cancelado")
	}
	if r.handle(0).has("end:timeout") || len(r.evs("call.ended")) != 0 {
		t.Fatalf("a chamada ativa não pode ser encerrada por timeout: %v", r.handle(0).calls())
	}
}

func TestStart_OriginatesAndTimesOut(t *testing.T) {
	r := newRig(t, false, nil)
	if err := r.s.Start(context.Background(), callA, "5511999990001@s.whatsapp.net"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "start", func() bool { return r.handle(0) != nil && r.handle(0).has("start") })
	if err := r.s.Start(context.Background(), callB, "5511999990002@s.whatsapp.net"); err != ErrBusy {
		t.Fatalf("segunda origem na mesma conexão: %v", err)
	}
	if r.clk.fire(ringTimeout) != 1 {
		t.Fatal("45 s de toque não armado")
	}
	eventually(t, "end:timeout", func() bool { return r.handle(0).has("end:timeout") })
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	e := r.evs("call.ended")[0]
	if e["direction"] != "outgoing" || e["endReason"] != "timeout" {
		t.Fatalf("call.ended: %v", e)
	}
}

func TestStart_Validation(t *testing.T) {
	r := newRig(t, false, nil)
	for _, c := range []struct{ id, to string }{
		{"curto", "5511999990001@s.whatsapp.net"},
		{callA, "120363000000000000@g.us"},
		{callA, "123@newsletter"},
		{callA, ""},
	} {
		if err := r.s.Start(context.Background(), c.id, c.to); err == nil {
			t.Fatalf("Start(%q,%q) devia falhar", c.id, c.to)
		}
	}
	if r.nHandles() != 0 {
		t.Fatal("entrada inválida não pode criar gerenciador")
	}
}

func TestStart_FailureFreesSlotAndPublishesFailed(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.newHandle = func(core.VoipSocket) Handle { return &failingStart{fakeHandle{}} }
	if err := r.s.Start(context.Background(), callA, "5511999990001@s.whatsapp.net"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "call.ended failed", func() bool {
		e := r.evs("call.ended")
		return len(e) == 1 && e[0]["endReason"] == ReasonFailed
	})
	if r.s.get(callA) != nil {
		t.Fatal("slot preso após falha")
	}
}

type failingStart struct{ fakeHandle }

func (f *failingStart) Start(context.Context, string, types.JID) error {
	return context.DeadlineExceeded
}

func TestCommands_ContractAndRejectOnlyByCommand(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })
	if r.handle(0).has("reject") {
		t.Fatal("reject não pode ocorrer sem comando")
	}
	if err := r.s.Reject(context.Background(), callA); err != nil {
		t.Fatal(err)
	}
	eventually(t, "reject por comando", func() bool { return r.handle(0).has("reject") })
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })

	for name, err := range map[string]error{
		"ready":  r.s.Ready(context.Background(), "ZZZ"),
		"accept": r.s.Accept(context.Background(), "ZZZ"),
		"reject": r.s.Reject(context.Background(), "ZZZ"),
		"end":    r.s.End(context.Background(), "ZZZ"),
	} {
		if err != ErrNoCall {
			t.Fatalf("%s de chamada inexistente: %v", name, err)
		}
	}
}

func TestAccept_RequiresReady(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	if err := r.s.Accept(context.Background(), callA); err != ErrNotRinging {
		t.Fatalf("atender sem call.ready: %v", err)
	}
}

func TestEnd_SendsTerminate(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	if err := r.s.End(context.Background(), callA); err != nil {
		t.Fatal(err)
	}
	eventually(t, "end", func() bool { return r.handle(0).has("end:user_ended") })
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	if r.evs("call.ended")[0]["durationSecs"] != 7 {
		t.Fatalf("%v", r.evs("call.ended"))
	}
}

func TestPanicIsolation_EndsOnlyThatCall(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.newHandle = func(core.VoipSocket) Handle { return &fakeHandle{panicOn: "offer"} }
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))

	eventually(t, "call.ended failed", func() bool {
		e := r.evs("call.ended")
		return len(e) == 1 && e[0]["endReason"] == ReasonFailed
	})
	if r.s.get(callA) != nil {
		t.Fatal("slot preso após pânico")
	}
	r.s.newHandle = func(core.VoipSocket) Handle { return &fakeHandle{} }
	r.s.OnOffer(context.Background(), offer(callB, pn("5511999990002")))
	if len(r.evs("call.incoming")) != 2 {
		t.Fatal("a conexão deve continuar atendendo depois do pânico")
	}
}

func TestOfferDoesNotBlockEventLoop(t *testing.T) {
	r := newRig(t, false, nil)
	gate := make(chan struct{})
	r.s.newHandle = func(core.VoipSocket) Handle { return &fakeHandle{started: gate} }

	done := make(chan struct{})
	go func() {
		r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("OnOffer bloqueou o laço de eventos enquanto a oferta aguardava o ack")
	}
	close(gate)
}

func TestAbandonAll(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })
	r.s.AbandonAll(ReasonInterrupted)
	e := r.evs("call.ended")
	if len(e) != 1 || e[0]["endReason"] != ReasonInterrupted {
		t.Fatalf("call.ended: %v", e)
	}
	if r.handle(0).has("reject") || r.handle(0).has("end:user_ended") {
		t.Fatalf("nada deve ser enviado ao WhatsApp: %v", r.handle(0).calls())
	}
}

func TestOffer_IgnoresEmptyCallID(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer("", pn("5511999990001")))
	if r.nHandles() != 0 || len(r.events) != 0 {
		t.Fatal("oferta sem id deve ser ignorada")
	}
}

func TestAnnounceReset_PublishesCallReset(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.AnnounceReset()
	e := r.evs("call.reset")
	if len(e) != 1 || e[0]["sessionId"] != "7" {
		t.Fatalf("call.reset: %v", e)
	}
}

func TestOfferNotice_GroupCallIsRegisteredAsUnsupportedAndNotAnswered(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOfferNotice(context.Background(), &events.CallOfferNotice{
		BasicCallMeta: types.BasicCallMeta{From: pn("5511999990001"), CallCreator: pn("5511999990001"), CallID: callA, GroupJID: types.NewJID("123", types.GroupServer)},
		Type:          "group", Media: "audio",
	})
	m := r.evs("call.missed")
	if len(m) != 1 || m[0]["reason"] != ReasonUnsupportedType || m[0]["callId"] != callA {
		t.Fatalf("call.missed: %v", m)
	}
	if r.nHandles() != 0 || len(r.evs("call.incoming")) != 0 {
		t.Fatal("aviso de grupo não pode abrir gerenciador nem tocar")
	}
}

// Contrato dos eventos que o business consome: nomes e campos não podem mudar
// sem mudar o consumidor (6.x).
func TestEventContract(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.tickEvery = 20 * time.Millisecond
	ctx := context.Background()
	r.s.OnOffer(ctx, offer(callA, pn("5511999990001")))
	r.s.OnOffer(ctx, offer(callB, pn("5511999990002")))
	eventually(t, "oferta", func() bool { return r.handle(0).has("offer") })
	_ = r.s.Ready(ctx, callA)
	r.handle(0).hooks.OnState(State{State: "connecting", Direction: "incoming"})
	_ = r.s.Accept(ctx, callA)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	eventually(t, "call.quality", func() bool { return len(r.evs("call.quality")) > 0 })
	_ = r.s.End(ctx, callA)
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })

	need := map[string][]string{
		"call.incoming": {"callId", "peer", "callerPn", "direction", "media", "sessionId"},
		"call.state":    {"callId", "peer", "callerPn", "direction", "state", "sessionId"},
		"call.ended":    {"callId", "peer", "callerPn", "direction", "endReason", "durationSecs", "sessionId"},
		"call.missed":   {"callId", "peer", "callerPn", "reason", "sessionId"},
		"call.quality":  {"callId", "lossPct", "jitterMs", "rttMs", "sessionId"},
	}
	for typ, fields := range need {
		evs := r.evs(typ)
		if len(evs) == 0 {
			t.Fatalf("nenhum evento %s foi publicado", typ)
		}
		for _, f := range fields {
			if _, ok := evs[0][f]; !ok {
				t.Errorf("%s sem o campo %q: %v", typ, f, evs[0])
			}
		}
	}
	if got := r.evs("call.incoming")[0]["direction"]; got != "incoming" {
		t.Errorf("direction=%v", got)
	}
	if got := r.evs("call.missed")[0]["reason"]; got != ReasonBusy {
		t.Errorf("reason=%v", got)
	}
}

// 9.5: se a saída UDP do servidor estiver bloqueada, o relay nunca conecta. A chamada
// não pode ficar presa em "Conectando…" para sempre: ao fim do prazo é encerrada como
// falha de mídia, com um motivo que o operador entende, e o slot da conexão é liberado.
func TestAccepted_MediaNeverConnects_EndsAsMediaTimeout(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	eventually(t, "accept", func() bool { return r.handle(0).has("accept") })

	if r.clk.fire(mediaConnectTimeout) != 1 {
		t.Fatal("o prazo de conexão da mídia não foi armado ao atender")
	}
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	e := r.evs("call.ended")[0]
	if e["endReason"] != ReasonMediaTimeout {
		t.Fatalf("endReason=%v, esperava %s", e["endReason"], ReasonMediaTimeout)
	}
	if !r.handle(0).has("end:" + ReasonMediaTimeout) {
		t.Fatalf("o contato precisa ser desconectado (terminate): %v", r.handle(0).calls())
	}
	if r.s.get(callA) != nil {
		t.Fatal("o slot da conexão não foi liberado")
	}
}

func TestAccepted_MediaConnects_CancelsTheMediaTimer(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	eventually(t, "accept", func() bool { return r.handle(0).has("accept") })

	r.handle(0).hooks.OnState(State{State: "active", Direction: "incoming"})
	if n := r.clk.fire(mediaConnectTimeout); n != 0 {
		t.Fatalf("a mídia conectou: o prazo não pode mais disparar (disparou %d)", n)
	}
	time.Sleep(30 * time.Millisecond)
	if len(r.evs("call.ended")) != 0 {
		t.Fatal("chamada ativa não pode ser encerrada pelo prazo de mídia")
	}
}

func TestStart_MediaNeverConnectsAfterContactAnswers_EndsAsMediaTimeout(t *testing.T) {
	r := newRig(t, false, nil)
	if err := r.s.Start(context.Background(), callA, "5511999990001@s.whatsapp.net"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "start", func() bool { return r.handle(0) != nil && r.handle(0).has("start") })
	// o contato atendeu: sai de "chamando" e entra em "conectando"
	r.handle(0).hooks.OnState(State{State: "connecting", Direction: "outgoing"})
	if r.clk.fire(mediaConnectTimeout) != 1 {
		t.Fatal("ao entrar em 'conectando' o prazo de mídia deve ser armado também na chamada originada")
	}
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	if got := r.evs("call.ended")[0]["endReason"]; got != ReasonMediaTimeout {
		t.Fatalf("endReason=%v", got)
	}
}
