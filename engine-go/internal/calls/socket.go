// Package calls liga a pilha VoIP portada (internal/voip) ao engine do Watink:
// adaptador do whatsmeow, ciclo de vida das chamadas, comandos e eventos.
// O engine é adaptador burro: nenhuma regra de negócio mora aqui.
package calls

import (
	"context"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/signaling"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

const defaultQueryTimeout = 15 * time.Second

// wire é o trecho de rede do whatsmeow que SendNode/Query usam. Existe só para
// o teste poder trocá-lo por um fake sem abrir uma sessão real.
type wire interface {
	SendNode(ctx context.Context, node waBinary.Node) error
	WaitResponse(id string) chan *waBinary.Node
	CancelResponse(id string, ch chan *waBinary.Node)
}

// Socket implementa core.VoipSocket sobre um *whatsmeow.Client.
type Socket struct {
	cli          *whatsmeow.Client
	w            wire
	queryTimeout time.Duration
}

var _ core.VoipSocket = (*Socket)(nil)

func NewSocket(cli *whatsmeow.Client) *Socket {
	return &Socket{cli: cli, w: cli.DangerousInternals(), queryTimeout: defaultQueryTimeout}
}

func (s *Socket) di() *whatsmeow.DangerousInternalClient { return s.cli.DangerousInternals() }

func (s *Socket) OwnPN() types.JID  { return s.di().GetOwnID() }
func (s *Socket) OwnLID() types.JID { return s.di().GetOwnLID() }

func (s *Socket) AccountDeviceIdentityNode() (waBinary.Node, bool) {
	if s.cli.Store == nil || s.cli.Store.Account == nil {
		return waBinary.Node{}, false
	}
	return s.di().MakeDeviceIdentityNode(), true
}

// SendNode envia o nó e respeita o cancelamento do ctx do chamador.
func (s *Socket) SendNode(ctx context.Context, node waBinary.Node) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.w.SendNode(ctx, node)
}

// Query envia o nó e espera a resposta com o mesmo id. Devolve (nil, nil) se o
// servidor não responder dentro do timeout (a pilha trata a ausência de ack),
// e o erro do ctx se ele for cancelado antes.
func (s *Socket) Query(ctx context.Context, node waBinary.Node) (*waBinary.Node, error) {
	id, _ := node.Attrs["id"].(string)
	if id == "" {
		return nil, s.SendNode(ctx, node)
	}
	ch := s.w.WaitResponse(id)
	if err := s.SendNode(ctx, node); err != nil {
		s.w.CancelResponse(id, ch)
		return nil, err
	}
	timer := time.NewTimer(s.queryTimeout)
	defer timer.Stop()
	select {
	case resp := <-ch:
		return resp, nil
	case <-timer.C:
		s.w.CancelResponse(id, ch)
		return nil, nil
	case <-ctx.Done():
		s.w.CancelResponse(id, ch)
		return nil, ctx.Err()
	}
}

func (s *Socket) GetUSyncDevices(ctx context.Context, jids []types.JID) ([]types.JID, error) {
	return s.cli.GetUserDevices(ctx, jids)
}

// AssertSessions não faz nada: o whatsmeow cria as sessões ao cifrar.
func (s *Socket) AssertSessions(context.Context, []types.JID, bool) error { return nil }

func (s *Socket) CreateParticipantNodes(ctx context.Context, devices []types.JID, callKey []byte, encAttrs waBinary.Attrs) ([]waBinary.Node, bool, error) {
	plaintext, err := signaling.EncodeCallKeyMessage(callKey)
	if err != nil {
		return nil, false, err
	}
	return s.di().EncryptMessageForDevices(ctx, devices, s.cli.GenerateMessageID(), plaintext, plaintext, encAttrs)
}

func (s *Socket) DecryptCallKey(ctx context.Context, from types.JID, encChild *waBinary.Node) ([]byte, error) {
	typ, _ := encChild.Attrs["type"].(string)
	plaintext, _, err := s.di().DecryptDM(ctx, encChild, from, typ == "pkmsg", time.Now())
	if err != nil {
		return nil, err
	}
	return signaling.DecodeCallKeyPlaintext(plaintext)
}

func (s *Socket) GetTCToken(ctx context.Context, jid types.JID) ([]byte, error) {
	if s.cli.Store == nil || s.cli.Store.PrivacyTokens == nil {
		return nil, nil
	}
	for _, cand := range []types.JID{s.ResolveLIDForPN(ctx, jid).ToNonAD(), jid.ToNonAD()} {
		if cand.IsEmpty() {
			continue
		}
		tok, err := s.cli.Store.PrivacyTokens.GetPrivacyToken(ctx, cand)
		if err != nil {
			return nil, err
		}
		if tok != nil && len(tok.Token) > 0 {
			return tok.Token, nil
		}
	}
	return nil, nil
}

func (s *Socket) ResolveLIDForPN(ctx context.Context, pn types.JID) types.JID {
	if pn.Server == types.HiddenUserServer {
		return pn
	}
	if s.cli.Store != nil && s.cli.Store.LIDs != nil {
		if lid, err := s.cli.Store.LIDs.GetLIDForPN(ctx, pn); err == nil && !lid.IsEmpty() {
			return lid
		}
	}
	if info, err := s.cli.GetUserInfo(ctx, []types.JID{pn}); err == nil {
		if lid := info[pn].LID; !lid.IsEmpty() {
			return lid
		}
	}
	return pn
}
