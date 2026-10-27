// Package services — message event handlers are split by event type:
//   - event_listener_msg_media.go    — handleMediaDownloaded
//   - event_listener_msg_ack.go      — handleMessageAck
//   - event_listener_msg_revoke.go   — handleMessageRevoke
//   - event_listener_msg_reaction.go — handleMessageReaction
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// handlePollVote records a contact's vote on a poll message.
// It only persists the result if the originating QuickAnswer has capture_results: true.
func (el *EventListener) handlePollVote(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	// O evento chega do barramento sem escopo de empresa: sem ele o RLS recusa a leitura e a gravação (ADR 0034).
	return tenantdb.WithTenant(ctx, el.db, tenantID, func(ctx context.Context) error {
		return el.recordPollVote(ctx, payload, tenantID)
	})
}

func (el *EventListener) recordPollVote(ctx context.Context, payload json.RawMessage, tenantID uuid.UUID) error {
	if el.db == nil {
		log.Printf("[handlePollVote] db is nil — cannot persist PollResult")
		return nil
	}

	var p PollVotePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	if p.PollMessageID == "" || p.VoterJID == "" {
		log.Printf("[handlePollVote] missing required fields: pollMessageId=%q voterJid=%q", p.PollMessageID, p.VoterJID)
		return nil
	}

	msg, err := el.messages.FindByID(ctx, p.PollMessageID, tenantID)
	if err != nil {
		log.Printf("[handlePollVote] message %s not found: %v", p.PollMessageID, err)
		return nil
	}
	if msg == nil {
		log.Printf("[handlePollVote] message %s not found (nil)", p.PollMessageID)
		return nil
	}

	// O voto do contato vira parte da enquete que o operador vê (independe de capture_results, que só governa PollResults).
	el.applyPollVote(ctx, msg, p)

	// If the message carries a quickAnswerId, check capture_results before persisting.
	if msg.DataJson != "" && msg.DataJson != "{}" {
		var msgData map[string]interface{}
		if err := json.Unmarshal([]byte(msg.DataJson), &msgData); err == nil {
			if rawQAID, ok := msgData["quickAnswerId"]; ok {
				var qaID float64
				switch v := rawQAID.(type) {
				case float64:
					qaID = v
				case json.Number:
					f, _ := v.Float64()
					qaID = f
				}

				if qaID > 0 {
					var qa models.QuickAnswer
					if err := el.db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).
						Where("id = ? AND \"tenantId\" = ?", int(qaID), tenantID).
						First(&qa).Error; err != nil {
						log.Printf("[handlePollVote] QuickAnswer %d not found: %v", int(qaID), err)
						return nil
					}

					var qaContent map[string]interface{}
					if err := json.Unmarshal([]byte(qa.Content), &qaContent); err != nil || qaContent == nil {
						return nil
					}
					captureRaw, exists := qaContent["capture_results"]
					if !exists {
						return nil
					}
					capture, _ := captureRaw.(bool)
					if !capture {
						return nil
					}
				}
			}
		}
	}

	result := models.PollResult{
		PollMessageID:  p.PollMessageID,
		ContactJID:     p.VoterJID,
		OptionSelected: p.OptionSelected,
		AnsweredAt:     time.Now(),
	}

	if err := el.db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).Create(&result).Error; err != nil {
		log.Printf("[handlePollVote] failed to create PollResult: %v", err)
		return err
	}

	log.Printf("[handlePollVote] recorded vote: msg=%s voter=%s option=%q", p.PollMessageID, p.VoterJID, p.OptionSelected)
	return nil
}

// pollOptionHashes mapeia o sha256 (hex) de cada opção ao seu nome: o WhatsApp só informa os hashes escolhidos.
func pollOptionHashes(options []string) map[string]string {
	m := make(map[string]string, len(options))
	for _, o := range options {
		h := sha256.Sum256([]byte(o))
		m[hex.EncodeToString(h[:])] = o
	}
	return m
}

// applyPollVote guarda em dataJson.votes[voterJid] as opções escolhidas (lista vazia = voto removido) e avisa a tela.
// Enquete sem opções estruturadas (anterior ao suporte) não tem como decodificar os hashes e é ignorada aqui.
func (el *EventListener) applyPollVote(ctx context.Context, msg *domain.Message, p PollVotePayload) {
	data := map[string]interface{}{}
	_ = json.Unmarshal([]byte(msg.DataJson), &data)
	rich, _ := data["rich"].(map[string]interface{})
	var options []string
	if raw, ok := rich["options"].([]interface{}); ok {
		for _, o := range raw {
			if str, ok := o.(string); ok {
				options = append(options, str)
			}
		}
	}
	if len(options) == 0 {
		return
	}
	byHash := pollOptionHashes(options)
	chosen := []string{}
	for _, h := range strings.Split(p.OptionSelected, ",") {
		if name, ok := byHash[strings.TrimSpace(h)]; ok {
			chosen = append(chosen, name)
		}
	}
	votes, _ := data["votes"].(map[string]interface{})
	if votes == nil {
		votes = map[string]interface{}{}
	}
	votes[p.VoterJID] = chosen
	data["votes"] = votes
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	if err := el.messages.Update(ctx, msg, map[string]interface{}{"dataJson": string(b)}); err != nil {
		log.Printf("[handlePollVote] não consegui gravar o voto em %s: %v", msg.ID, err)
		return
	}
	msg.DataJson = string(b)
	el.bcast().EmitToRoom("/", "chat:"+strconv.Itoa(msg.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": msg})
}
