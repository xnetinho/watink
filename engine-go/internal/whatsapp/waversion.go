package whatsapp

import (
	"context"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"

	"github.com/alltomatos/watinkdev/engine-go/internal/safego"
)

// RefreshWAVersion resolve a versão do cliente WhatsApp (site oficial, registro do WPPConnect, WATINK_WA_VERSION, embutida).
// Chamado na subida do engine, antes de qualquer sessão conectar.
func (s *WhatsAppService) RefreshWAVersion(ctx context.Context) {
	if s.waVersion != nil {
		s.waVersion.Refresh(ctx)
	}
}

// handleClientOutdated trata o 405 do WhatsApp: renova a versão e reconecta a sessão UMA vez por versão. Se a versão nova
// também for recusada, para (a sessão fica DISCONNECTED) em vez de martelar o servidor.
func (s *WhatsAppService) handleClientOutdated(client *whatsmeow.Client, id int, tenantID string) {
	s.emitStatus(id, tenantID, "DISCONNECTED")
	if s.waVersion == nil {
		return
	}
	safego.Go("clientOutdated", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.waVersion.Refresh(ctx)
		s.reconnectOnNewVersion(id, client.Connect)
	})
}

// reconnectOnNewVersion reconecta só se ainda não tentou com a versão vigente. Devolve se tentou.
func (s *WhatsAppService) reconnectOnNewVersion(id int, connect func() error) bool {
	cur := store.GetWAVersion()
	s.waVersionMu.Lock()
	tried := s.waRetried[id] == cur
	s.waRetried[id] = cur
	s.waVersionMu.Unlock()
	if tried {
		log.Printf("Session %d: o WhatsApp recusou a versão %s e não há outra mais nova; sem nova tentativa", id, cur)
		return false
	}
	log.Printf("Session %d: versão recusada (405); reconectando com %s", id, cur)
	if err := connect(); err != nil {
		log.Printf("Session %d: falha ao reconectar após atualizar a versão: %v", id, err)
	}
	return true
}

func (s *WhatsAppService) clearOutdatedRetry(id int) {
	s.waVersionMu.Lock()
	delete(s.waRetried, id)
	s.waVersionMu.Unlock()
}
