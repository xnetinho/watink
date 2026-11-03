package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/store"
)

func TestReconnectOnNewVersion_UmaVezPorVersao(t *testing.T) {
	s := &WhatsAppService{waRetried: map[int]store.WAVersionContainer{}}
	n := 0
	connect := func() error { n++; return nil }
	if !s.reconnectOnNewVersion(1, connect) || s.reconnectOnNewVersion(1, connect) || n != 1 {
		t.Fatalf("segunda tentativa com a mesma versão deveria ser barrada (n=%d)", n)
	}
	if !s.reconnectOnNewVersion(2, connect) || n != 2 {
		t.Fatalf("outra sessão tem tentativa própria (n=%d)", n)
	}
	s.clearOutdatedRetry(1)
	if !s.reconnectOnNewVersion(1, connect) || n != 3 {
		t.Fatalf("após conectar com sucesso a sessão volta a poder tentar (n=%d)", n)
	}
}

func TestHandleClientOutdated_SemResolverSoMarcaDesconectada(t *testing.T) {
	s, calls := newTestService()
	s.handleClientOutdated(nil, 1, "t")
	if len(*calls) != 1 || (*calls)[0].eventType != "session.status" {
		t.Fatalf("esperava um session.status, veio %v", *calls)
	}
}
