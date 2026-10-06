package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/alltomatos/watinkdev/engine-go/internal/calls"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ErrCallsUnavailable: a sessão não tem chamadas (não conectada ou não iniciada).
var ErrCallsUnavailable = errors.New("sessão sem suporte a chamadas")

func (s *WhatsAppService) callSession(id int) *calls.Session {
	s.callMu.Lock()
	defer s.callMu.Unlock()
	return s.callSessions[id]
}

// resetCalls (re)cria o estado de chamadas de uma sessão quando o cliente é
// (re)iniciado. proxied vem do proxyUrl: com proxy não há chamada (ADR 0021).
// Uma chamada que sobrou do cliente anterior é abandonada.
func (s *WhatsAppService) resetCalls(id int, tenantID string, client *whatsmeow.Client, proxied bool) {
	sess := calls.NewSession(calls.SessionConfig{
		ID: id, TenantID: tenantID, Sock: calls.NewSocket(client), Proxied: proxied,
		Publish: s.publishEvent,
		LookupPN: func(ctx context.Context, lid types.JID) (types.JID, error) {
			if client.Store == nil || client.Store.LIDs == nil {
				return types.EmptyJID, nil
			}
			return client.Store.LIDs.GetPNForLID(ctx, lid)
		},
	})
	s.callMu.Lock()
	old := s.callSessions[id]
	s.callSessions[id] = sess
	s.callMu.Unlock()
	if old != nil {
		old.AbandonAll(calls.ReasonInterrupted)
	}
	sess.AnnounceReset()
}

// dropCalls encerra localmente a chamada da sessão (parada, logout, queda).
func (s *WhatsAppService) dropCalls(id int) {
	s.callMu.Lock()
	sess := s.callSessions[id]
	s.callMu.Unlock()
	if sess != nil {
		sess.AbandonAll(calls.ReasonInterrupted)
	}
}

// handleCallEvent roteia os eventos de chamada do whatsmeow. Nada aqui bloqueia o
// laço de eventos: a Session despacha o trabalho pesado em goroutine.
func (s *WhatsAppService) handleCallEvent(id int, evt interface{}) {
	sess := s.callSession(id)
	if sess == nil {
		return
	}
	ctx := context.Background()
	switch v := evt.(type) {
	case *events.CallOffer:
		sess.OnOffer(ctx, v)
	case *events.CallAccept:
		sess.OnAccept(ctx, v)
	case *events.CallTransport:
		sess.OnTransport(ctx, v)
	case *events.CallRelayLatency:
		sess.OnRelayLatency(v)
	case *events.CallTerminate:
		sess.OnTerminate(v)
	case *events.CallReject:
		sess.OnReject(v)
	case *events.CallOfferNotice:
		sess.OnOfferNotice(ctx, v)
	case *events.UnknownCallEvent:
		log.Printf("Session %d: evento de chamada desconhecido ignorado", id)
	}
}

// CallPayload é o corpo dos comandos call.*.
type CallPayload struct {
	CallID string `json:"callId"`
	To     string `json:"to"`
}

// HandleCallCommand executa um comando call.* (ready, accept, reject, end, start).
func (s *WhatsAppService) HandleCallCommand(sessionID int, cmd string, p CallPayload) error {
	sess := s.callSession(sessionID)
	if sess == nil {
		return fmt.Errorf("%w: session %d", ErrCallsUnavailable, sessionID)
	}
	ctx := context.Background()
	switch cmd {
	case "call.ready":
		return sess.Ready(ctx, p.CallID)
	case "call.accept":
		return sess.Accept(ctx, p.CallID)
	case "call.reject":
		return sess.Reject(ctx, p.CallID)
	case "call.end":
		return sess.End(ctx, p.CallID)
	case "call.start":
		if _, err := s.getConnectedClient(sessionID); err != nil {
			return err
		}
		return sess.Start(ctx, p.CallID, p.To)
	}
	log.Printf("Unknown call command: %s", cmd)
	return nil
}

// OpenCallAudio abre o canal de áudio da chamada, procurando-a em todas as
// sessões (o callId é único). Usado pelo endpoint interno de áudio.
func (s *WhatsAppService) OpenCallAudio(callID string) (*calls.AudioPipe, error) {
	s.callMu.Lock()
	sessions := make([]*calls.Session, 0, len(s.callSessions))
	for _, sess := range s.callSessions {
		sessions = append(sessions, sess)
	}
	s.callMu.Unlock()
	for _, sess := range sessions {
		p, err := sess.OpenAudio(callID)
		if errors.Is(err, calls.ErrNoCall) {
			continue
		}
		return p, err
	}
	return nil, calls.ErrNoCall
}

// CallsLoad resume a carga de chamadas para o /health.
func (s *WhatsAppService) CallsLoad() (active int, audioQueued int) {
	s.callMu.Lock()
	sessions := make([]*calls.Session, 0, len(s.callSessions))
	for _, sess := range s.callSessions {
		sessions = append(sessions, sess)
	}
	s.callMu.Unlock()
	for _, sess := range sessions {
		a, q := sess.Load()
		active += a
		audioQueued += q
	}
	return
}
