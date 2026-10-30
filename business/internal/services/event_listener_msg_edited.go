package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
)

// handleMessageEdited aplica a edição de uma mensagem do WhatsApp: guarda a versão na cadeia e atualiza o texto atual.
// A edição feita POR NÓS pelo aparelho (fromMe) também chega aqui, vinda do celular pareado, e é registrada como nossa.
func (el *EventListener) handleMessageEdited(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	var p MessageEditedPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	if p.MessageID == "" {
		return nil
	}
	editedAt, err := time.Parse(time.RFC3339Nano, p.EditedAt)
	if err != nil {
		editedAt = time.Now()
	}
	by := models.EditedByContact
	if p.FromMe {
		by = models.EditedByMe
	}

	res, err := RecordEdit(ctx, el.db, RecordEditInput{TenantID: tenantID, MessageID: p.MessageID, Body: p.Body, EditedAt: editedAt, EditedBy: by})
	if errors.Is(err, ErrEditTargetNotFound) {
		// A original não está no Watink (mensagem anterior à conexão, ou apagada): não há o que atualizar.
		log.Printf("[handleMessageEdited] original %s não encontrada; edição ignorada", p.MessageID)
		return nil
	}
	if errors.Is(err, ErrEmptyEdit) {
		log.Printf("[handleMessageEdited] edição de %s sem texto; ignorada para não esvaziar a mensagem", p.MessageID)
		return nil
	}
	if err != nil {
		return err
	}
	if res.Duplicate {
		return nil // reentrega do mesmo evento: já foi aplicada e anunciada
	}
	el.bcast().EmitToRoom("/", "chat:"+strconv.Itoa(res.Message.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": res.Message})
	return nil
}
