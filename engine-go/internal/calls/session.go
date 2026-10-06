package calls

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

var callIDPattern = regexp.MustCompile(`^[A-F0-9]{32}$`)

// Session guarda as chamadas de UMA conexão do WhatsApp: no máximo uma ativa.
//
// Regra de ouro: s.mu nunca fica travado enquanto se chama o Handle. O
// CallManager dispara os hooks com o mutex dele travado, e os hooks pegam s.mu;
// segurar s.mu ao chamar o Handle inverteria a ordem dos locks.
type Session struct {
	id       int
	tenantID string

	sock      core.VoipSocket
	newHandle NewHandleFunc
	publish   PublishFunc
	clock     clock
	proxied   bool
	lookupPN  func(ctx context.Context, lid types.JID) (types.JID, error)

	mu     sync.Mutex
	active *activeCall
}

type activeCall struct {
	id        string
	h         Handle
	incoming  bool
	peer      string
	callerPn  string
	ready     bool
	accepted  bool
	ended     bool
	state     string
	offerDone chan struct{}
	timer     stopper
}

// SessionConfig reúne o que uma Session precisa; Clock e NewHandle são
// opcionais (produção usa os reais).
type SessionConfig struct {
	ID        int
	TenantID  string
	Sock      core.VoipSocket
	Publish   PublishFunc
	Proxied   bool
	LookupPN  func(ctx context.Context, lid types.JID) (types.JID, error)
	NewHandle NewHandleFunc
	Clock     clock
}

func NewSession(c SessionConfig) *Session {
	if c.NewHandle == nil {
		c.NewHandle = NewManagerHandle
	}
	if c.Clock == nil {
		c.Clock = realClock{}
	}
	return &Session{
		id: c.ID, tenantID: c.TenantID, sock: c.Sock, newHandle: c.NewHandle,
		publish: c.Publish, clock: c.Clock, proxied: c.Proxied, lookupPN: c.LookupPN,
	}
}

func (s *Session) emit(eventType string, p map[string]interface{}) {
	p["sessionId"] = fmt.Sprintf("%d", s.id)
	s.publish(s.tenantID, s.id, eventType, p)
}

func (s *Session) missed(callID, peer, callerPn, reason string) {
	s.emit("call.missed", map[string]interface{}{
		"callId": callID, "peer": peer, "callerPn": callerPn, "reason": reason,
	})
}

func (s *Session) get(callID string) *activeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil && s.active.id == callID {
		return s.active
	}
	return nil
}

// register instala a chamada como a ativa. Devolve false se já há outra.
func (s *Session) register(ac *activeCall) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		return false
	}
	s.active = ac
	return true
}

func (s *Session) setTimer(ac *activeCall, t stopper) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ac.timer != nil {
		ac.timer.Stop()
	}
	ac.timer = t
}

// hooks liga o gerenciador de uma chamada ao publicador de eventos.
func (s *Session) hooks(ac *activeCall) Hooks {
	return Hooks{
		OnState: func(st State) {
			if st.State == string(core.CallStateEnded) {
				s.finish(ac, st)
				return
			}
			s.mu.Lock()
			ac.state = st.State
			s.mu.Unlock()
			s.emit("call.state", map[string]interface{}{"callId": ac.id, "state": st.State, "direction": st.Direction})
		},
		OnEnded: func(st State) { s.finish(ac, st) },
	}
}

// finish encerra o registro da chamada uma única vez: libera o slot, para o
// temporizador e publica call.ended.
func (s *Session) finish(ac *activeCall, st State) {
	s.mu.Lock()
	if ac.ended {
		s.mu.Unlock()
		return
	}
	ac.ended = true
	if ac.timer != nil {
		ac.timer.Stop()
		ac.timer = nil
	}
	if s.active == ac {
		s.active = nil
	}
	s.mu.Unlock()

	dir := "outgoing"
	if ac.incoming {
		dir = "incoming"
	}
	s.emit("call.ended", map[string]interface{}{
		"callId": ac.id, "peer": ac.peer, "callerPn": ac.callerPn, "direction": dir,
		"endReason": st.EndReason, "durationSecs": st.DurationSecs,
	})
}

// run executa fn num gerenciador protegido: um pânico no codec ou no transporte
// encerra só esta chamada (call.ended "failed"), nunca o engine.
func (s *Session) run(ac *activeCall, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			s.finish(ac, State{CallID: ac.id, EndReason: ReasonFailed})
		}
	}()
	fn()
}

func wrapCall(from types.JID, inner *waBinary.Node) *waBinary.Node {
	content := []waBinary.Node{}
	if inner != nil {
		content = append(content, *inner)
	}
	return &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"from": from}, Content: content}
}

func hasChild(n *waBinary.Node, tag string) bool {
	if n == nil {
		return false
	}
	kids, _ := n.Content.([]waBinary.Node)
	for i := range kids {
		if kids[i].Tag == tag {
			return true
		}
	}
	return false
}
