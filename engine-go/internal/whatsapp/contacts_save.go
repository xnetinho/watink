package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// buildContactSavePatch monta o patch de app state que grava um contato na agenda do celular principal, o mesmo
// recurso que o WhatsApp Web usa ao criar um contato (índice "contact", patch critical_unblock_low).
func buildContactSavePatch(jid types.JID, fullName string) appstate.PatchInfo {
	fullName = strings.TrimSpace(fullName)
	first := fullName
	if i := strings.IndexByte(fullName, ' '); i > 0 {
		first = fullName[:i]
	}
	return appstate.PatchInfo{
		Type: appstate.WAPatchCriticalUnblockLow,
		Mutations: []appstate.MutationInfo{{
			Index:   []string{appstate.IndexContact, jid.String()},
			Version: 2,
			Value: &waSyncAction.SyncActionValue{
				ContactAction: &waSyncAction.ContactAction{
					FullName:                 proto.String(fullName),
					FirstName:                proto.String(first),
					SaveOnPrimaryAddressbook: proto.Bool(true),
				},
			},
		}},
	}
}

// SaveContact grava um contato criado no Watink na agenda do celular da conexão. Só números com WhatsApp podem ser
// gravados: um número sem conta é recusado, sem fingir sucesso.
func (s *WhatsAppService) SaveContact(sessionID int, tenantID string, payload SaveContactPayload) error {
	err := s.saveContact(sessionID, tenantID, payload)
	result := map[string]interface{}{"sessionId": fmt.Sprintf("%d", sessionID), "number": strings.TrimPrefix(strings.TrimSpace(payload.Number), "+"), "ok": err == nil}
	if err != nil {
		result["error"] = err.Error()
	}
	s.publishEvent(tenantID, sessionID, "contact.saved", result)
	return err
}

func (s *WhatsAppService) saveContact(sessionID int, tenantID string, payload SaveContactPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		return err
	}
	number := strings.TrimPrefix(strings.TrimSpace(payload.Number), "+")
	if number == "" || strings.TrimSpace(payload.Name) == "" {
		return fmt.Errorf("contact.save: nome e número são obrigatórios")
	}

	ctx := context.Background()
	res, err := client.IsOnWhatsApp(ctx, []string{"+" + number})
	if err != nil {
		return fmt.Errorf("contact.save: consultar %s: %w", number, err)
	}
	if len(res) == 0 || !res[0].IsIn {
		return fmt.Errorf("contact.save: %s não tem WhatsApp", number)
	}

	if err := client.SendAppState(ctx, buildContactSavePatch(res[0].JID, payload.Name)); err != nil {
		return fmt.Errorf("contact.save: gravar na agenda do celular: %w", err)
	}
	return nil
}
