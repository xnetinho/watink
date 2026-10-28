package whatsapp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// Política de presença do número (por conexão). "Disponível" faz o WhatsApp parar de notificar o celular do operador e
// mostrar o número como online ao contato; é também o que faz o WhatsApp entregar a presença dos contatos.
const (
	policyNever    = "never"
	policyOnDemand = "on_demand"
	policyAlways   = "always"

	defaultIdleMinutes = 5
)

// presenceSender é o que a política precisa do cliente WhatsApp; separado para testar sem rede.
type presenceSender interface {
	SendPresence(ctx context.Context, state types.Presence) error
}

// presenceState guarda a política e o estado de uma sessão. Todos os acessos passam por mu.
type presenceState struct {
	mu        sync.Mutex
	policy    string
	idle      time.Duration
	available bool        // o que já enviamos ao WhatsApp
	timer     *time.Timer // volta a indisponível depois de idle sem operador (on_demand)
}

func normalizePolicy(p string) string {
	switch p {
	case policyOnDemand, policyAlways:
		return p
	}
	return policyNever
}

func idleDuration(minutes int) time.Duration {
	if minutes < 1 {
		minutes = defaultIdleMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func (s *WhatsAppService) presenceFor(id int) *presenceState {
	s.presenceMu.Lock()
	defer s.presenceMu.Unlock()
	if s.presence == nil {
		s.presence = map[int]*presenceState{}
	}
	st, ok := s.presence[id]
	if !ok {
		st = &presenceState{policy: policyNever, idle: idleDuration(0)}
		s.presence[id] = st
	}
	return st
}

// setAvailable envia disponível/indisponível só se mudou (cada envio é uma operação no WhatsApp). Chamar com st.mu preso.
func (st *presenceState) setAvailable(c presenceSender, want bool) error {
	if st.available == want {
		return nil
	}
	state := types.PresenceUnavailable
	if want {
		state = types.PresenceAvailable
	}
	if err := c.SendPresence(context.Background(), state); err != nil {
		return err
	}
	st.available = want
	return nil
}

// applyPolicy aplica uma política nova a uma sessão já conectada.
func (st *presenceState) applyPolicy(c presenceSender, policy string, idleMinutes int) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.policy = normalizePolicy(policy)
	st.idle = idleDuration(idleMinutes)
	if st.timer != nil {
		st.timer.Stop()
		st.timer = nil
	}
	switch st.policy {
	case policyAlways:
		return st.setAvailable(c, true)
	case policyNever:
		return st.setAvailable(c, false)
	default: // on_demand: só fica disponível enquanto há operador; ao trocar para cá, começa indisponível
		return st.setAvailable(c, false)
	}
}

// onConnect reaplica a política quando a sessão (re)conecta: o WhatsApp esquece a presença ao cair a conexão.
func (st *presenceState) onConnect(c presenceSender) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.available = false // a conexão nova começa indisponível do ponto de vista do servidor
	if st.policy == policyAlways {
		return st.setAvailable(c, true)
	}
	return nil
}

// operatorActive é chamado quando um operador abre/mantém uma conversa da conexão. Só age em on_demand: marca disponível
// e (re)arma o temporizador que volta a indisponível. Devolve true se o número ficou/está disponível.
func (st *presenceState) operatorActive(c presenceSender, onIdle func()) (bool, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	switch st.policy {
	case policyAlways:
		return true, st.setAvailable(c, true)
	case policyOnDemand:
		if err := st.setAvailable(c, true); err != nil {
			return false, err
		}
		if st.timer != nil {
			st.timer.Stop()
		}
		st.timer = time.AfterFunc(st.idle, onIdle)
		return true, nil
	}
	return false, nil
}

// goIdle volta a indisponível (on_demand, sem operador há idle).
func (st *presenceState) goIdle(c presenceSender) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.policy != policyOnDemand {
		return nil
	}
	st.timer = nil
	return st.setAvailable(c, false)
}

// shouldMarkAvailable diz se vale mandar "disponível" antes de assinar a presença de um contato.
func (st *presenceState) shouldMarkAvailable() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.policy == policyOnDemand || st.policy == policyAlways
}

// ApplyPresencePolicy muda a política de uma sessão em andamento (comando presence.policy).
func (s *WhatsAppService) ApplyPresencePolicy(sessionID int, tenantID string, p PresencePolicyPayload) error {
	st := s.presenceFor(sessionID)
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		// Sessão fora do ar: só guarda; onConnect aplica ao conectar.
		st.mu.Lock()
		st.policy, st.idle = normalizePolicy(p.Policy), idleDuration(p.IdleMinutes)
		st.mu.Unlock()
		return nil
	}
	return st.applyPolicy(client, p.Policy, p.IdleMinutes)
}

// presenceOnConnect é chamado ao conectar a sessão.
func (s *WhatsAppService) presenceOnConnect(sessionID int, client presenceSender) {
	if err := s.presenceFor(sessionID).onConnect(client); err != nil {
		log.Printf("[Session-%d] presença ao conectar: %v", sessionID, err)
	}
}

// preparePresence marca disponível, se a política permitir, e arma o temporizador. Chamado antes de assinar um contato.
func (s *WhatsAppService) preparePresence(sessionID int, client presenceSender) error {
	st := s.presenceFor(sessionID)
	if !st.shouldMarkAvailable() {
		return nil
	}
	_, err := st.operatorActive(client, func() {
		if err := st.goIdle(client); err != nil {
			log.Printf("[Session-%d] voltar a indisponível: %v", sessionID, err)
		}
	})
	if err != nil {
		return fmt.Errorf("marcar disponível: %w", err)
	}
	return nil
}
