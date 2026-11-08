package calls

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
)

// Estados do registro da chamada.
const (
	StatusRinging     = "ringing"
	StatusActive      = "active"
	StatusEnded       = "ended"
	StatusMissed      = "missed"
	StatusRejected    = "rejected"
	StatusFailed      = "failed"
	StatusInterrupted = "interrupted"
)

// IncomingEvent é o corpo de call.incoming e call.missed publicado pelo engine.
type IncomingEvent struct {
	CallID    string `json:"callId"`
	SessionID string `json:"sessionId"`
	Peer      string `json:"peer"`
	CallerPn  string `json:"callerPn"`
	Direction string `json:"direction"`
	Reason    string `json:"reason"`
	// Media: audio | video. O engine de antes do vídeo não manda o campo: vale audio.
	Media string `json:"media"`
	// Platform e Version: o aparelho que o chamador declarou no <offer>. Vazios quando o engine é anterior a este campo.
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

// callerPlatforms são as plataformas que o protocolo do WhatsApp declara (whatsmeow, PatchDebugData_Platform) e que já vimos chegar.
// Só elas são gravadas: o valor vem do aparelho do chamador e nunca pode virar texto livre no banco nem na tela.
var callerPlatforms = map[string]bool{
	"android": true, "iphone": true, "web": true, "smba": true, "smbi": true, "ipad": true, "darwin": true, "uwp": true,
}

// NormalizeCallerPlatform devolve a plataforma em minúsculas se for conhecida, ou "" (desconhecida, vazia ou lixo).
func NormalizeCallerPlatform(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if callerPlatforms[p] {
		return p
	}
	return ""
}

// normalizeCallerVersion mantém só o que parece versão (dígitos e pontos, até 32): é texto do aparelho do chamador.
func normalizeCallerVersion(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 32 {
		return ""
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '.' {
			return ""
		}
	}
	return v
}

// mediaOf normaliza o tipo de mídia do evento: só "video" é vídeo, qualquer outra coisa é voz.
func mediaOf(m string) string {
	if m == "video" {
		return "video"
	}
	return "audio"
}

// StateEvent é o corpo de call.state.
type StateEvent struct {
	CallID    string `json:"callId"`
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	Direction string `json:"direction"`
}

// EndedEvent é o corpo de call.ended.
type EndedEvent struct {
	CallID       string `json:"callId"`
	SessionID    string `json:"sessionId"`
	Peer         string `json:"peer"`
	CallerPn     string `json:"callerPn"`
	Direction    string `json:"direction"`
	EndReason    string `json:"endReason"`
	DurationSecs int    `json:"durationSecs"`
}

// QualityEvent é o corpo de call.quality.
type QualityEvent struct {
	CallID      string   `json:"callId"`
	SessionID   string   `json:"sessionId"`
	RttMs       *float64 `json:"rttMs"`
	LossPct     float64  `json:"lossPct"`
	JitterMs    float64  `json:"jitterMs"`
	NoPeerAudio bool     `json:"noPeerAudio"`
	SilentMs    int64    `json:"silentMs"`
}

func parseSession(raw string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(raw))
	return n
}

// ResolveContactAndTicket acha o contato do chamador (unificando LID e telefone)
// e o ticket aberto da conexão; sem ticket aberto, cria um pendente. Devolve o
// ticket para o registro da chamada aparecer no histórico da conversa.
func (s *Service) ResolveContactAndTicket(ctx context.Context, tenantID uuid.UUID, whatsappID int, peer, callerPn string) (*domain.Contact, *domain.Ticket, error) {
	isLID := strings.HasSuffix(peer, "@lid")
	number := callerPn
	if number == "" {
		number = peerNumber(peer)
	}
	contact, err := s.contacts.FindOrCreate(ctx, tenantID, whatsappID, number, "", "", false, isLID, peer, callerPn)
	if err != nil {
		return nil, nil, err
	}
	convKey := domain.WhatsAppConversationKey(contact.ID)
	ticket, err := s.tickets.FindOpenByConversation(ctx, tenantID, whatsappID, convKey)
	if err != nil {
		return nil, nil, err
	}
	if ticket == nil {
		var queueID *int
		if s.queues != nil {
			if ids, qerr := s.queues.FindQueueIDsByChannel(ctx, whatsappID, tenantID); qerr == nil && len(ids) == 1 {
				queueID = &ids[0]
			}
		}
		ticket, err = s.tickets.FindOrCreatePending(ctx, &domain.Ticket{
			ContactID: contact.ID, Status: "pending", TenantID: tenantID, ChannelConnectionID: whatsappID, ConversationKey: convKey, QueueID: queueID,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return contact, ticket, nil
}

// upsertLog grava a chamada pela chave (tenantId, callId). Reentrega do mesmo
// evento não duplica: o conflito não faz nada e o registro existente é devolvido.
func (s *Service) upsertLog(ctx context.Context, tenantID uuid.UUID, l *models.CallLog) (*models.CallLog, bool, error) {
	l.TenantID = tenantID
	res := s.fresh(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenantId"}, {Name: "callId"}},
		DoNothing: true,
	}).Create(l)
	if res.Error != nil {
		return nil, false, res.Error
	}
	created := res.RowsAffected > 0
	var got models.CallLog
	if err := s.fresh(ctx).Where(`"tenantId" = ? AND "callId" = ?`, tenantID, l.CallID).First(&got).Error; err != nil {
		return nil, false, err
	}
	return &got, created, nil
}

// HandleIncoming trata call.incoming: registra a chamada, decide quem toca e
// confirma ao engine (call.ready) só se houver alguém para atender. Sem elegível
// não confirma: o engine não envia nada e o celular segue tocando.
func (s *Service) HandleIncoming(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev IncomingEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	waID := parseSession(ev.SessionID)
	if ev.CallID == "" || waID == 0 {
		return fmt.Errorf("call.incoming sem callId/sessionId")
	}
	contact, ticket, err := s.ResolveContactAndTicket(ctx, tenantID, waID, ev.Peer, ev.CallerPn)
	if err != nil {
		return err
	}
	connID := ticket.ChannelConnectionID
	l, created, err := s.upsertLog(ctx, tenantID, &models.CallLog{
		CallID: ev.CallID, ChannelConnectionID: connID, ContactID: &contact.ID, TicketID: &ticket.ID,
		Direction: "incoming", Media: mediaOf(ev.Media), Status: StatusRinging, PeerJid: ev.Peer, CallerPn: ev.CallerPn, StartedAt: s.now(),
		CallerPlatform: NormalizeCallerPlatform(ev.Platform), CallerVersion: normalizeCallerVersion(ev.Version),
	})
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	hasPerm := func(userID int) bool { return s.userCan(ctx, tenantID, userID, "receive") }
	users, err := s.Eligible(ctx, tenantID, waID, hasPerm)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return s.markMissed(ctx, tenantID, l, "no_operator")
	}
	if err := s.command(ctx, tenantID, waID, "call.ready", map[string]interface{}{"callId": ev.CallID}); err != nil {
		return err
	}
	payload := s.callPayload(l, contact, ticket)
	for _, u := range users {
		s.bcast.EmitToRoom("/", UserRoom(tenantID, u.ID), "call.incoming", payload)
	}
	return nil
}

// userCan consulta Users e as permissões do cargo, que têm RLS: o handle precisa herdar o escopo de empresa do ctx. Com o pool
// global a leitura era recusada e a função respondia "sem permissão" para todo mundo, então nenhuma chamada tocava.
func (s *Service) userCan(ctx context.Context, tenantID uuid.UUID, userID int, action string) bool {
	return auth.UserHasPermission(s.fresh(ctx), userID, tenantID, "calls", action)
}

// command pede ao canal da conexão que execute a operação de chamada (call.accept, call.reject, call.end, call.start,
// call.ready). O canal precisa saber controlar chamadas; senão a chamada é recusada em vez de ficar no ar.
func (s *Service) command(ctx context.Context, tenantID uuid.UUID, connectionID int, cmd string, payload map[string]interface{}) error {
	var conn models.ChannelConnection
	// s.fresh herda o escopo de empresa do ctx (RLS); o pool global recusa a leitura de ChannelConnections.
	if err := s.fresh(ctx).
		Where(`id = ? AND "tenantId" = ?`, connectionID, tenantID).First(&conn).Error; err != nil {
		return fmt.Errorf("conexão %d não encontrada: %w", connectionID, err)
	}
	ch, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind))
	if err != nil {
		return err
	}
	cc, ok := ch.(domain.CallControl)
	if !ok {
		return fmt.Errorf("o canal %q não faz chamadas", conn.Kind)
	}
	return cc.CallCommand(ctx, &conn, cmd, payload)
}

func (s *Service) callPayload(l *models.CallLog, c *domain.Contact, t *domain.Ticket) map[string]interface{} {
	return map[string]interface{}{
		"callId": l.CallID, "connectionId": l.ChannelConnectionID, "direction": l.Direction, "status": l.Status,
		"media": l.Media, "contact": c, "ticketId": l.TicketID, "startedAt": l.StartedAt, "platform": l.CallerPlatform,
	}
}

// HandleMissed trata call.missed (oferta ignorada pelo engine).
func (s *Service) HandleMissed(ctx context.Context, tenantID uuid.UUID, raw json.RawMessage) error {
	var ev IncomingEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	waID := parseSession(ev.SessionID)
	if ev.CallID == "" || waID == 0 {
		return fmt.Errorf("call.missed sem callId/sessionId")
	}
	contact, ticket, err := s.ResolveContactAndTicket(ctx, tenantID, waID, ev.Peer, ev.CallerPn)
	if err != nil {
		return err
	}
	l, _, err := s.upsertLog(ctx, tenantID, &models.CallLog{
		CallID: ev.CallID, ChannelConnectionID: waID, ContactID: &contact.ID, TicketID: &ticket.ID,
		Direction: "incoming", Status: StatusMissed, PeerJid: ev.Peer, CallerPn: ev.CallerPn,
		StartedAt: s.now(), EndReason: ev.Reason,
		CallerPlatform: NormalizeCallerPlatform(ev.Platform), CallerVersion: normalizeCallerVersion(ev.Version),
	})
	if err != nil {
		return err
	}
	if l.Status == StatusMissed && l.EndedAt != nil {
		return nil
	}
	return s.markMissed(ctx, tenantID, l, ev.Reason)
}

// markMissed encerra o registro como perdido, escreve a mensagem de sistema no
// ticket e avisa os operadores que o toque acabou. Só age uma vez por chamada.
func (s *Service) markMissed(ctx context.Context, tenantID uuid.UUID, l *models.CallLog, reason string) error {
	now := s.now()
	res := s.fresh(ctx).Model(&models.CallLog{}).
		Where(`"tenantId" = ? AND "callId" = ? AND "endedAt" IS NULL`, tenantID, l.CallID).
		Updates(map[string]interface{}{"status": StatusMissed, "endReason": reason, "endedAt": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	l.Status, l.EndReason, l.EndedAt = StatusMissed, reason, &now
	s.writeSystemMessage(ctx, tenantID, l)
	s.notifyEnded(tenantID, l)
	return nil
}

// notifyEnded manda call.ended ao tenant para o toque sumir em todo navegador.
func (s *Service) notifyEnded(tenantID uuid.UUID, l *models.CallLog) {
	s.bcast.EmitToTenantRoom(tenantID.String(), "call.ended", map[string]interface{}{
		"callId": l.CallID, "status": l.Status, "endReason": l.EndReason,
		"durationSec": l.DurationSec, "ticketId": l.TicketID,
	})
}
