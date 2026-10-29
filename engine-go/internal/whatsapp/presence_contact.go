package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// SubscribePresence pede ao WhatsApp as atualizações de presença de um contato. O servidor só informa se o contato
// permite (privacidade): sem permissão não chega nada, e isso não é erro.
func (s *WhatsAppService) SubscribePresence(sessionID int, tenantID string, p PresenceSubscribePayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		return err
	}
	jid, err := ensureJID(p.To)
	if err != nil {
		return fmt.Errorf("invalid JID %q: %w", p.To, err)
	}
	if jid.Server == types.GroupServer {
		return nil // grupo não tem presença
	}
	// O WhatsApp só entrega a presença de quem está "disponível": a política da conexão decide se e por quanto tempo
	// (ver presence_policy.go). Em never, nada é marcado e só chega presença se o número já estiver ativo por outro meio.
	if err := s.preparePresence(sessionID, client); err != nil {
		return err
	}
	return client.SubscribePresence(context.Background(), jid.ToNonAD())
}

// handlePresenceEvent publica online/offline e, quando o contato permite, a última vez visto.
func (s *WhatsAppService) handlePresenceEvent(id int, tenantID string, v *events.Presence) {
	payload := map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", id),
		"jid":       v.From.ToNonAD().String(),
		"online":    !v.Unavailable,
	}
	if !v.LastSeen.IsZero() {
		payload["lastSeen"] = v.LastSeen.Unix()
	}
	s.publishEvent(tenantID, id, "contact.presence", payload)
}
