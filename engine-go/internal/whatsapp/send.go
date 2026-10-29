package whatsapp

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// SendText sends a plain text message, with optional quoted reply and mentions.
func (s *WhatsAppService) SendText(sessionID int, tenantID string, payload TextCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}

	to, err := ensureJID(payload.To)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return fmt.Errorf("invalid JID %q: %w", payload.To, err)
	}

	msg := buildTextMessage(payload.Body, payload.QuotedMsgID, payload.QuotedJID, payload.QuotedBody, payload.QuotedMedia, payload.Mentions, forwardInfo(payload.Forwarded, payload.ForwardScore))
	_, err = client.SendMessage(context.Background(), to, msg, whatsmeow.SendRequestExtra{ID: types.MessageID(payload.MessageID)})
	if err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.send", err)
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}
	s.emitAck(sessionID, tenantID, payload.MessageID, 1)
	return nil
}

// SendMedia uploads and sends an image/video/audio/document message.
func (s *WhatsAppService) SendMedia(sessionID int, tenantID string, payload MediaCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}

	to, err := ensureJID(payload.To)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return fmt.Errorf("invalid JID %q: %w", payload.To, err)
	}

	data, err := resolveMediaBytes(payload)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}

	// Miniatura, dimensões, duração e onda: o que o cliente oficial manda junto e uma biblioteca costuma omitir (sem a
	// miniatura o celular do destinatário mostra a mídia "vazia" até baixá-la). Melhor esforço: falha aqui não barra o envio.
	var meta mediameta.Meta
	data, payload, meta = enrichMedia(context.Background(), data, payload)
	log.Printf("[media] %s tipo=%s bytes=%d mime=%q miniatura=%dB %dx%d duração=%ds páginas=%d onda=%d barras ffmpeg=%v poppler=%v",
		payload.MessageID, payload.MediaType, len(data), payload.MimeType, len(meta.Thumbnail), max(meta.Width, meta.ThumbW), max(meta.Height, meta.ThumbH), meta.Seconds, meta.Pages, len(meta.Waveform), mediameta.Available(), mediameta.AvailablePDF())

	mediaType := normalizeMediaType(payload.MediaType)
	uploaded, err := client.Upload(context.Background(), data, mediaType)
	if err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.upload", err)
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}

	message := buildMediaMessage(payload, uploaded, meta)
	_, err = client.SendMessage(context.Background(), to, message, whatsmeow.SendRequestExtra{ID: types.MessageID(payload.MessageID)})
	if err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.send", err)
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}
	s.emitAck(sessionID, tenantID, payload.MessageID, 1)
	return nil
}

// SendPresence sets the chat-composing indicator ("digitando...") for one
// chat. Best-effort by design (business already treats this as fire-and-
// forget pacing, not a delivery guarantee) — unlike SendText/SendMedia this
// never emits an ack, since there's no per-message ID to correlate.
func (s *WhatsAppService) SendPresence(sessionID int, tenantID string, payload PresenceCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		return err
	}

	chat, err := ensureJID(payload.To)
	if err != nil {
		return fmt.Errorf("invalid JID %q: %w", payload.To, err)
	}

	state := types.ChatPresencePaused
	if payload.State == "composing" {
		state = types.ChatPresenceComposing
	}

	return client.SendChatPresence(context.Background(), chat, state, types.ChatPresenceMediaText)
}

// MarkRead marks one or more messages as read for the given chat.
func (s *WhatsAppService) MarkRead(sessionID int, tenantID string, payload MarkReadCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		return err
	}

	chat, err := ensureJID(payload.ChatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID %q: %w", payload.ChatJID, err)
	}
	sender := chat
	if payload.SenderJID != "" {
		if parsed, parseErr := ensureJID(payload.SenderJID); parseErr == nil {
			sender = parsed
		}
	}

	ids := make([]types.MessageID, 0, len(payload.MessageIDs))
	for _, id := range payload.MessageIDs {
		ids = append(ids, types.MessageID(id))
	}
	err = client.MarkRead(context.Background(), ids, time.Now(), chat, sender)
	if err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.markread", err)
	}
	return err
}

// SendReaction sends (or, when payload.Reaction == "", removes) an emoji
// reaction to an existing message.
func (s *WhatsAppService) SendReaction(sessionID int, tenantID string, payload ReactionCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}

	chat, err := ensureJID(payload.To)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return fmt.Errorf("invalid JID %q: %w", payload.To, err)
	}

	sender := chat
	if payload.TargetFromMe && client.Store.ID != nil {
		sender = *client.Store.ID
	}

	msg := client.BuildReaction(chat, sender, types.MessageID(payload.TargetMsgID), payload.Reaction)
	_, err = client.SendMessage(context.Background(), chat, msg, whatsmeow.SendRequestExtra{ID: types.MessageID(payload.MessageID)})
	if err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.reaction", err)
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}
	s.emitAck(sessionID, tenantID, payload.MessageID, 1)
	return nil
}
