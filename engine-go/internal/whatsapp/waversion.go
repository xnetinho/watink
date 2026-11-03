package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"

	"github.com/alltomatos/watinkdev/engine-go/internal/safego"
	"github.com/alltomatos/watinkdev/engine-go/internal/waversion"
)

// loadWAVersionSetting lê a escolha do superadmin. O search_path do engine é "whatsmeow", então a tabela é qualificada.
// Sem linha (instalação que nunca abriu a tela) vale o modo automático.
func loadWAVersionSetting(db *sql.DB) func(ctx context.Context) (waversion.Setting, error) {
	return func(ctx context.Context) (waversion.Setting, error) {
		var mode string
		var manual sql.NullString
		err := db.QueryRowContext(ctx, `SELECT "mode", "manualVersion" FROM public."WaVersionSettings" WHERE id = 1`).Scan(&mode, &manual)
		if errors.Is(err, sql.ErrNoRows) {
			return waversion.Setting{Mode: waversion.ModeAuto}, nil
		}
		if err != nil {
			return waversion.Setting{}, err
		}
		return waversion.Setting{Mode: mode, Manual: manual.String}, nil
	}
}

// RefreshWAVersion resolve a versão do cliente WhatsApp conforme a tela do superadmin. Na subida do engine, antes de
// qualquer sessão conectar, e ao salvar na tela (force). Sessões já conectadas não caem; a nova versão vale na próxima conexão.
func (s *WhatsAppService) RefreshWAVersion(ctx context.Context, force bool) bool {
	if s.waVersion == nil {
		return false
	}
	changed := s.waVersion.Refresh(ctx, force)
	if changed {
		safego.Go("reconnectRefused", s.reconnectRefused)
	}
	return changed
}

// refusedSessions lista as sessões que levaram 405 e esperam uma versão nova.
func (s *WhatsAppService) refusedSessions() []int {
	s.waVersionMu.Lock()
	defer s.waVersionMu.Unlock()
	ids := make([]int, 0, len(s.waRetried))
	for id := range s.waRetried {
		ids = append(ids, id)
	}
	return ids
}

// reconnectRefused reconecta as sessões recusadas quando a versão muda (por exemplo, o superadmin corrigiu a versão manual).
func (s *WhatsAppService) reconnectRefused() {
	for _, id := range s.refusedSessions() {
		s.mu.RLock()
		client := s.clients[id]
		s.mu.RUnlock()
		if client != nil {
			s.reconnectOnNewVersion(id, reconnectFn(client))
		}
	}
}

// WAVersionStatus devolve as versões lado a lado (embutida, oficial, registro, vigente) e a escolha atual.
func (s *WhatsAppService) WAVersionStatus(ctx context.Context) (waversion.Status, error) {
	if s.waVersion == nil {
		return waversion.Status{}, errors.New("resolvedor de versão indisponível")
	}
	return s.waVersion.Status(ctx), nil
}

// handleClientOutdated trata o 405 do WhatsApp: marca a sessão desconectada, renova a versão e reconecta UMA vez por versão.
// Se a versão (inclusive a manual) também for recusada, para em vez de martelar o servidor.
func (s *WhatsAppService) handleClientOutdated(client *whatsmeow.Client, id int, tenantID string) {
	s.emitStatus(id, tenantID, "DISCONNECTED")
	if s.waVersion == nil {
		return
	}
	safego.Go("clientOutdated", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.waVersion.Refresh(ctx, false)
		s.reconnectOnNewVersion(id, reconnectFn(client))
	})
}

// reconnectFn reconecta o cliente. O 405 chega com o socket ainda aberto e o Connect() recusa ("already connected"): nesse caso
// fecha antes e conecta de novo.
func reconnectFn(client *whatsmeow.Client) func() error {
	return func() error { return reconnect(client.Connect, client.Disconnect) }
}

func reconnect(connect func() error, disconnect func()) error {
	err := connect()
	if errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		disconnect()
		return connect()
	}
	return err
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
