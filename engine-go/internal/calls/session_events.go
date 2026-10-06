package calls

import (
	"context"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// callerPN devolve o telefone (só dígitos) do chamador. O WhatsApp pode
// identificar o chamador por um LID; nesse caso tenta o alternativo e, por fim,
// o mapa LID↔telefone do store. Vazio se desconhecido.
func (s *Session) callerPN(ctx context.Context, creator, alt types.JID) string {
	for _, j := range []types.JID{creator, alt} {
		if j.Server == types.DefaultUserServer && j.User != "" {
			return j.User
		}
	}
	if s.lookupPN != nil {
		for _, j := range []types.JID{creator, alt} {
			if j.Server != types.HiddenUserServer || j.User == "" {
				continue
			}
			if pn, err := s.lookupPN(ctx, j.ToNonAD()); err == nil && pn.User != "" {
				return pn.User
			}
		}
	}
	return ""
}

func creatorOf(m types.BasicCallMeta) types.JID {
	if !m.CallCreator.IsEmpty() {
		return m.CallCreator.ToNonAD()
	}
	return m.From.ToNonAD()
}

// OnOffer trata uma oferta de chamada recebida. Nunca recusa: sem condição de
// atender, a oferta é ignorada e o celular segue tocando.
func (s *Session) OnOffer(ctx context.Context, evt *events.CallOffer) {
	callID := evt.CallID
	if callID == "" {
		return
	}
	creator := creatorOf(evt.BasicCallMeta)
	peer := creator.String()
	pn := s.callerPN(ctx, evt.CallCreator, evt.CallCreatorAlt)

	switch {
	case !evt.GroupJID.IsEmpty() || hasChild(evt.Data, "video"):
		s.missed(callID, peer, pn, ReasonUnsupportedType)
		return
	case s.proxied:
		s.missed(callID, peer, pn, ReasonProxyBlocked)
		return
	}

	ac := &activeCall{id: callID, incoming: true, peer: peer, callerPn: pn, offerDone: make(chan struct{})}
	if !s.register(ac) {
		s.missed(callID, peer, pn, ReasonBusy)
		return
	}
	ac.h = s.newHandle(s.sock)
	ac.h.SetHooks(s.hooks(ac))
	s.wireMedia(ac)
	s.setTimer(ac, s.clock.AfterFunc(readyTimeout, func() { s.onReadyTimeout(ac) }))

	// O processamento da oferta (decifrar chave, ler relays) sai do laço de
	// eventos do whatsmeow. Eventos seguintes da mesma chamada esperam offerDone.
	go func() {
		defer close(ac.offerDone)
		s.run(ac, func() { ac.h.HandleOffer(ctx, wrapCall(evt.From, evt.Data), evt.From) })
	}()
	s.emit("call.incoming", map[string]interface{}{
		"callId": callID, "peer": peer, "callerPn": pn, "direction": "incoming", "media": "audio",
	})
}

// OnOfferNotice trata o aviso de oferta que o WhatsApp manda para chamadas em
// grupo (e vídeo anunciado por aviso). Nunca se atende nem se recusa: só se
// registra "tipo não suportado" e o celular segue tocando.
func (s *Session) OnOfferNotice(ctx context.Context, evt *events.CallOfferNotice) {
	if evt.CallID == "" {
		return
	}
	creator := creatorOf(evt.BasicCallMeta)
	s.missed(evt.CallID, creator.String(), s.callerPN(ctx, evt.CallCreator, evt.CallCreatorAlt), ReasonUnsupportedType)
}

func (s *Session) onReadyTimeout(ac *activeCall) {
	s.mu.Lock()
	skip := ac.ended || ac.ready
	s.mu.Unlock()
	if skip {
		return
	}
	s.run(ac, func() { ac.h.Abandon(ReasonNoOperator) })
	s.missed(ac.id, ac.peer, ac.callerPn, ReasonNoOperator)
}

func (s *Session) onRingTimeout(ac *activeCall) {
	s.run(ac, func() {
		if ac.incoming {
			ac.h.Abandon(ReasonTimeout)
			return
		}
		_ = ac.h.End(context.Background(), ReasonTimeout)
	})
}

// afterOffer roda fn depois de a oferta ser processada, fora do laço de eventos.
func (s *Session) afterOffer(ac *activeCall, fn func()) {
	go func() {
		<-ac.offerDone
		s.run(ac, fn)
	}()
}

func (s *Session) OnAccept(ctx context.Context, evt *events.CallAccept) {
	ac := s.get(evt.CallID)
	if ac == nil {
		return
	}
	s.afterOffer(ac, func() {
		s.mu.Lock()
		elsewhere := ac.incoming && !ac.accepted
		s.mu.Unlock()
		if elsewhere {
			ac.h.Abandon(ReasonAcceptedElse)
			return
		}
		ac.h.HandleAccept(ctx, wrapCall(evt.From, evt.Data), evt.From)
	})
}

func (s *Session) OnTransport(ctx context.Context, evt *events.CallTransport) {
	if ac := s.get(evt.CallID); ac != nil {
		s.afterOffer(ac, func() { ac.h.HandleTransport(ctx, wrapCall(evt.From, evt.Data), evt.From) })
	}
}

func (s *Session) OnRelayLatency(evt *events.CallRelayLatency) {
	if ac := s.get(evt.CallID); ac != nil {
		s.afterOffer(ac, func() { ac.h.HandleRelayLatency(wrapCall(evt.From, evt.Data)) })
	}
}

func (s *Session) OnTerminate(evt *events.CallTerminate) {
	if ac := s.get(evt.CallID); ac != nil {
		s.afterOffer(ac, func() { ac.h.HandleTerminate(wrapCall(evt.From, evt.Data)) })
	}
}

// OnReject: o contato (ou outro aparelho) recusou. Encerra como terminate.
func (s *Session) OnReject(evt *events.CallReject) {
	if ac := s.get(evt.CallID); ac != nil {
		s.afterOffer(ac, func() { ac.h.HandleTerminate(wrapCall(evt.From, evt.Data)) })
	}
}
