package services

import (
	"testing"
	"time"
)

func TestPresenceCache(t *testing.T) {
	now := time.Now()
	RememberPresence("t1", "5511", PresenceInfo{Online: false, LastSeen: 1793132972, At: now.Add(-48 * time.Hour).Unix()})
	if p, ok := LastPresence("t1", "5511", now); !ok || p.LastSeen != 1793132972 {
		t.Fatalf("última vez visto não envelhece: %+v %v", p, ok)
	}
	RememberPresence("t1", "5522", PresenceInfo{Online: true, At: now.Add(-10 * time.Minute).Unix()})
	if _, ok := LastPresence("t1", "5522", now); ok {
		t.Fatal("online antigo vira desconhecido (o contato pode ter saído sem aviso)")
	}
	RememberPresence("t1", "5533", PresenceInfo{Online: true, At: now.Add(-30 * time.Second).Unix()})
	if p, ok := LastPresence("t1", "5533", now); !ok || !p.Online {
		t.Fatalf("online recente vale: %+v %v", p, ok)
	}
}

func TestPresenceCache_IsolaEmpresas(t *testing.T) {
	RememberPresence("empresa-a", "5599", PresenceInfo{Online: true})
	if _, ok := LastPresence("empresa-b", "5599", time.Now()); ok {
		t.Fatal("a presença de uma empresa não pode vazar para outra")
	}
}
