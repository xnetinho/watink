package services

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/mediawait"
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/google/uuid"
)

// handleMediaDownloaded consumes the result of an on-demand media download from
// the engine: it persists the bytes, points the message at the stored file and
// flips its status to "downloaded", then notifies the frontend so the blurred
// placeholder is replaced by the real media.
func (el *EventListener) handleMediaDownloaded(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	messages := el.messages
	var p struct {
		MessageID string `json:"messageId"`
		MediaData string `json:"mediaData"`
		Mimetype  string `json:"mimetype"`
		Error     string `json:"error"`
		// MediaType e ViewOnce vêm quando a mídia foi RECUPERADA de uma visualização única citada pelo celular principal.
		MediaType string `json:"mediaType"`
		ViewOnce  bool   `json:"viewOnce"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	if p.MessageID == "" {
		return nil
	}

	// Entrega direto para quem estiver aguardando este messageID via
	// mediawait.Waiter (ex.: AssistantRuntime transcrevendo um áudio) — os
	// bytes crus, sem depender do mediastore/DB abaixo terem terminado. Um
	// no-op quando ninguém está esperando (download normal via botão da UI).
	if el.mediaWaiter != nil {
		el.mediaWaiter.Fulfill(p.MessageID, mediawait.Result{MediaData: p.MediaData, MimeType: p.Mimetype, Err: p.Error})
	}

	msg, err := messages.FindByID(ctx, p.MessageID, tenantID)
	if err != nil || msg == nil {
		return nil
	}

	data := map[string]interface{}{}
	_ = json.Unmarshal([]byte(msg.DataJson), &data)

	emitFailed := func() error {
		data["mediaStatus"] = "failed"
		newData, _ := json.Marshal(data)
		_ = messages.Update(ctx, msg, map[string]interface{}{"dataJson": string(newData)})
		msg.DataJson = string(newData)
		el.bcast().EmitToRoom("/", "chat:"+strconv.Itoa(msg.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": msg})
		return nil
	}

	if p.Error != "" || p.MediaData == "" {
		return emitFailed()
	}

	mimeType := p.Mimetype
	if mimeType == "" {
		if mt, ok := data["mimetype"].(string); ok {
			mimeType = mt
		}
	}

	mediaURL, err := mediastore.SaveMediaBase64(p.MediaData, mimeType)
	if err != nil {
		log.Printf("[EventListener] media download save failed (msg %s): %v", p.MessageID, err)
		return emitFailed()
	}

	data["mediaStatus"] = "downloaded"
	// Drop the heavy serialized proto now that the media is stored locally.
	delete(data, "mediaProto")
	newData, _ := json.Marshal(data)

	upd := map[string]interface{}{"mediaUrl": mediaURL, "dataJson": string(newData)}
	// Visualização única recuperada: o aviso (view_once) vira a mídia de verdade e passa a ser desenhada como mídia comum.
	// "recoveredViewOnce" fica no dataJson só para a origem não se perder (nada na tela depende dele).
	if p.ViewOnce && msg.MediaType == "view_once" && p.MediaType != "" {
		upd["mediaType"] = p.MediaType
		data["recoveredViewOnce"] = true
		data["mimetype"] = mimeType
		recovered, _ := json.Marshal(data)
		upd["dataJson"] = string(recovered)
		newData = recovered
		msg.MediaType = p.MediaType
	}
	if err := messages.Update(ctx, msg, upd); err != nil {
		return err
	}
	msg.MediaUrl = mediaURL
	msg.DataJson = string(newData)

	if p.ViewOnce && p.MediaType != "" {
		el.refreshTicketPreviewAfterRecovery(ctx, msg, tenantID)
	}

	el.bcast().EmitToRoom("/", "chat:"+strconv.Itoa(msg.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": msg})
	return nil
}

// refreshTicketPreviewAfterRecovery atualiza a prévia da lista de conversas quando a mensagem recuperada ainda é a última do
// ticket: a prévia dizia "Visualização única" e agora a mensagem é uma foto. Se já chegou algo depois, não mexe.
func (el *EventListener) refreshTicketPreviewAfterRecovery(ctx context.Context, msg *domain.Message, tenantID uuid.UUID) {
	if el.tickets == nil {
		return
	}
	ticket, err := el.tickets.FindByID(ctx, msg.TicketID, tenantID)
	if err != nil || ticket == nil || ticket.LastMessageType != "view_once" {
		return
	}
	text, kind := domain.LastMessagePreview(msg.MediaType, msg.Body, "")
	if err := el.tickets.Update(ctx, ticket, map[string]interface{}{"lastMessage": text, "lastMessageType": kind}); err != nil {
		log.Printf("[handleMediaDownloaded] prévia do ticket %d não atualizada: %v", ticket.ID, err)
		return
	}
	ticket.LastMessage, ticket.LastMessageType = text, kind
	el.bcast().EmitToTenantRoom(tenantID.String(), "ticket", map[string]interface{}{"action": "update", "ticket": ticket})
}
