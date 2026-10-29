package services

import (
	"sync"
	"time"
)

// PresenceInfo é o último estado conhecido de presença de um contato.
type PresenceInfo struct {
	Online   bool  `json:"online"`
	LastSeen int64 `json:"lastSeen,omitempty"`
	At       int64 `json:"at"` // quando soubemos (unix), para descartar o que envelheceu
}

// presenceTTL: o WhatsApp só avisa quando a presença MUDA; ao assinar não devolve o estado atual. Guardamos o último
// recebido para mostrar ao abrir a conversa. "online" envelhece rápido (o contato pode ter saído sem aviso); a última
// vez visto, não.
const presenceOnlineTTL = 2 * time.Minute

type presenceCache struct {
	mu sync.RWMutex
	m  map[string]PresenceInfo // chave: tenant|número
}

// shortcut: em memória de um processo, perde-se ao reiniciar e não é compartilhado entre instâncias; mover para o Redis
// quando houver mais de uma instância do business.
var presences = &presenceCache{m: map[string]PresenceInfo{}}

func presenceKey(tenant, number string) string { return tenant + "|" + number }

// RememberPresence guarda o estado recebido do engine.
func RememberPresence(tenant, number string, p PresenceInfo) {
	if p.At == 0 {
		p.At = time.Now().Unix()
	}
	presences.mu.Lock()
	presences.m[presenceKey(tenant, number)] = p
	presences.mu.Unlock()
}

// LastPresence devolve o último estado, tratando "online" muito antigo como desconhecido.
func LastPresence(tenant, number string, now time.Time) (PresenceInfo, bool) {
	presences.mu.RLock()
	p, ok := presences.m[presenceKey(tenant, number)]
	presences.mu.RUnlock()
	if !ok {
		return PresenceInfo{}, false
	}
	if p.Online && now.Sub(time.Unix(p.At, 0)) > presenceOnlineTTL {
		return PresenceInfo{}, false
	}
	return p, true
}
