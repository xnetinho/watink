package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types"
)

// VotePoll envia o voto do operador em uma enquete. O voto é cifrado com o segredo da enquete original, que o
// whatsmeow guardou ao recebê-la ("Stored message secret key"); sem ele o WhatsApp não aceita o voto.
func (s *WhatsAppService) VotePoll(sessionID int, tenantID string, p PollVoteCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		return err
	}
	chat, err := ensureJID(p.Chat)
	if err != nil {
		return fmt.Errorf("chat inválido %q: %w", p.Chat, err)
	}
	sender, err := ensureJID(p.PollSender)
	if err != nil {
		return fmt.Errorf("remetente da enquete inválido %q: %w", p.PollSender, err)
	}
	if p.PollID == "" {
		return fmt.Errorf("pollId é obrigatório")
	}
	info := &types.MessageInfo{
		MessageSource: types.MessageSource{Chat: chat, Sender: sender, IsFromMe: p.PollFromMe, IsGroup: chat.Server == types.GroupServer},
		ID:            types.MessageID(p.PollID),
	}
	ctx := context.Background()
	vote, err := client.BuildPollVote(ctx, info, p.Options)
	if err != nil {
		return fmt.Errorf("montar voto: %w", err)
	}
	if _, err := client.SendMessage(ctx, chat, vote); err != nil {
		return fmt.Errorf("enviar voto: %w", err)
	}
	return nil
}
