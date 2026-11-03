package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types/events"
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

func TestHandleEvent_ClientOutdatedMarcaDesconectada(t *testing.T) {
	s, calls := newTestService()
	s.clients[1] = &whatsmeow.Client{}
	s.handleEvent(1, "t", &events.ClientOutdated{})
	if len(*calls) != 1 || (*calls)[0].eventType != "session.status" {
		t.Fatalf("o evento 405 deve chegar ao tratador e marcar a sessão: %v", *calls)
	}
}

func TestHandleEvent_ConnectedLimpaOContadorDe405(t *testing.T) {
	s, _ := newTestService()
	s.waRetried = map[int]store.WAVersionContainer{1: {2, 3000, 1}}
	s.clearOutdatedRetry(1)
	if _, ok := s.waRetried[1]; ok {
		t.Fatal("contador deveria ser limpo ao conectar")
	}
}

func TestRefusedSessions_ListaQuemLevou405(t *testing.T) {
	s := &WhatsAppService{waRetried: map[int]store.WAVersionContainer{}}
	if len(s.refusedSessions()) != 0 {
		t.Fatal("sem 405, ninguém é listado")
	}
	s.reconnectOnNewVersion(3, func() error { return nil })
	s.reconnectOnNewVersion(5, func() error { return nil })
	if got := s.refusedSessions(); len(got) != 2 {
		t.Fatalf("esperava 2 sessões, veio %v", got)
	}
	s.clearOutdatedRetry(3)
	if got := s.refusedSessions(); len(got) != 1 || got[0] != 5 {
		t.Fatalf("após conectar, só a 5 fica na lista: %v", got)
	}
}

func TestReconnect_SocketAindaAbertoFechaEConectaDeNovo(t *testing.T) {
	var steps []string
	first := true
	connect := func() error {
		steps = append(steps, "connect")
		if first {
			first = false
			return whatsmeow.ErrAlreadyConnected
		}
		return nil
	}
	if err := reconnect(connect, func() { steps = append(steps, "disconnect") }); err != nil {
		t.Fatal(err)
	}
	if got := len(steps); got != 3 || steps[1] != "disconnect" {
		t.Fatalf("esperava connect, disconnect, connect: %v", steps)
	}
}

func TestReconnect_ErroDiferenteNaoFecha(t *testing.T) {
	disconnected := false
	err := reconnect(func() error { return errors.New("rede") }, func() { disconnected = true })
	if err == nil || disconnected {
		t.Fatalf("erro comum deve voltar sem fechar: err=%v disconnected=%v", err, disconnected)
	}
}
