package calls

import (
	"context"
	"errors"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/signaling"
	"go.mau.fi/whatsmeow/types"
)

var (
	ErrNoCall     = errors.New("chamada não encontrada")
	ErrBusy       = errors.New("a conexão já tem uma chamada ativa")
	ErrProxied    = errors.New("conexão com proxy não faz chamadas")
	ErrBadPeer    = errors.New("destinatário inválido")
	ErrBadCallID  = errors.New("callId inválido")
	ErrNotRinging = errors.New("a chamada não está tocando")
)

// Ready é o comando call.ready: o business confirmou operador elegível. Só aqui
// o engine envia `preaccept` e passa a contar os 45 s de toque.
func (s *Session) Ready(ctx context.Context, callID string) error {
	ac := s.get(callID)
	if ac == nil {
		return ErrNoCall
	}
	s.mu.Lock()
	if ac.ended || !ac.incoming {
		s.mu.Unlock()
		return ErrNotRinging
	}
	already := ac.ready
	ac.ready = true
	s.mu.Unlock()
	if already {
		return nil
	}
	s.setTimer(ac, s.clock.AfterFunc(ringTimeout, func() { s.onRingTimeout(ac) }))
	s.afterOffer(ac, func() { _ = ac.h.SendPreaccept(ctx) })
	return nil
}

// Accept é o comando call.accept. O business garante que só um operador chega aqui.
func (s *Session) Accept(ctx context.Context, callID string) error {
	ac := s.get(callID)
	if ac == nil {
		return ErrNoCall
	}
	s.mu.Lock()
	if ac.ended || !ac.incoming || !ac.ready {
		s.mu.Unlock()
		return ErrNotRinging
	}
	ac.accepted = true
	s.mu.Unlock()
	s.setTimer(ac, nil)
	go func() {
		<-ac.offerDone
		s.run(ac, func() { _ = ac.h.Accept(ctx, callID) })
	}()
	return nil
}

// Reject é o comando call.reject: o ÚNICO caminho em que o engine envia `reject`.
func (s *Session) Reject(ctx context.Context, callID string) error {
	ac := s.get(callID)
	if ac == nil {
		return ErrNoCall
	}
	go func() {
		<-ac.offerDone
		s.run(ac, func() { _ = ac.h.Reject(ctx, callID) })
	}()
	return nil
}

// End é o comando call.end (operador desliga ou cancela a chamada).
func (s *Session) End(ctx context.Context, callID string) error {
	ac := s.get(callID)
	if ac == nil {
		return ErrNoCall
	}
	go s.run(ac, func() { _ = ac.h.End(ctx, "user_ended") })
	return nil
}

// Start é o comando call.start: origina uma chamada de voz ao telefone `to`.
// O callId vem do business (para ele já ter o registro antes do toque).
func (s *Session) Start(ctx context.Context, callID, to string) error {
	if callID == "" {
		callID = signaling.GenerateCallID()
	} else if !callIDPattern.MatchString(callID) {
		return ErrBadCallID
	}
	if s.proxied {
		return ErrProxied
	}
	peer, err := types.ParseJID(to)
	if err != nil || peer.IsEmpty() || peer.Server == types.GroupServer || peer.Server == types.NewsletterServer {
		return ErrBadPeer
	}
	ac := &activeCall{id: callID, peer: peer.String(), offerDone: closedChan()}
	if !s.register(ac) {
		return ErrBusy
	}
	ac.h = s.newHandle(s.sock)
	ac.h.SetHooks(s.hooks(ac))
	s.setTimer(ac, s.clock.AfterFunc(ringTimeout, func() { s.onRingTimeout(ac) }))
	go s.run(ac, func() {
		if err := ac.h.Start(ctx, callID, peer); err != nil {
			s.finish(ac, State{CallID: callID, EndReason: ReasonFailed})
		}
	})
	return nil
}

func closedChan() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

// AbandonAll encerra localmente toda chamada em andamento (desconexão da
// sessão ou parada do engine), sem enviar nada ao WhatsApp.
func (s *Session) AbandonAll(reason string) {
	s.mu.Lock()
	ac := s.active
	s.mu.Unlock()
	if ac != nil && ac.h != nil {
		s.run(ac, func() { ac.h.Abandon(reason) })
		s.finish(ac, State{CallID: ac.id, EndReason: reason})
	}
}
