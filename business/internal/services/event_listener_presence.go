package services

import (
	"encoding/json"

	"github.com/google/uuid"
)

// handleContactPresence repassa a presença (online/última vez visto) à tela. É efêmera: não grava nada, porque ela
// envelhece em segundos e só interessa a quem está olhando a conversa naquele instante.
func (el *EventListener) handleContactPresence(payload json.RawMessage, tenantID uuid.UUID) error {
	var p struct {
		JID      string `json:"jid"`
		Online   bool   `json:"online"`
		LastSeen int64  `json:"lastSeen"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	if p.JID == "" {
		return nil
	}
	RememberPresence(tenantID.String(), jidToNumber(p.JID), PresenceInfo{Online: p.Online, LastSeen: p.LastSeen})
	el.bcast().EmitToTenantRoom(tenantID.String(), "contactPresence", map[string]interface{}{
		"jid":      p.JID,
		"number":   jidToNumber(p.JID),
		"online":   p.Online,
		"lastSeen": p.LastSeen,
	})
	return nil
}
