package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func mustJID(t *testing.T, s string) types.JID {
	t.Helper()
	j, err := types.ParseJID(s)
	if err != nil {
		t.Fatalf("ParseJID(%q): %v", s, err)
	}
	return j
}

func TestResolveChatPN_LIDChatResolvedByStoreMap(t *testing.T) {
	lid := mustJID(t, "163423740493865@lid")
	pn := mustJID(t, "558382341576@s.whatsapp.net")
	client := newClientWithLIDStore(t, &fakeLIDStore{pnForLID: map[string]types.JID{lid.String(): pn}})

	got := resolveChatPN(client, types.MessageInfo{MessageSource: types.MessageSource{Chat: lid, Sender: lid}})
	if got != "558382341576@s.whatsapp.net" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveChatPN_NoStoreMapFallsBackToSenderAlt(t *testing.T) {
	lid := mustJID(t, "163423740493865@lid")
	client := newClientWithLIDStore(t, &fakeLIDStore{})

	info := types.MessageInfo{MessageSource: types.MessageSource{
		Chat: lid, Sender: lid, SenderAlt: mustJID(t, "558382341576@s.whatsapp.net"),
	}}
	if got := resolveChatPN(client, info); got != "558382341576@s.whatsapp.net" {
		t.Fatalf("recebida: got %q", got)
	}
}

func TestResolveChatPN_FromMeUsesRecipientAltNotSenderAlt(t *testing.T) {
	lid := mustJID(t, "163423740493865@lid")
	client := newClientWithLIDStore(t, &fakeLIDStore{})

	// Mensagem enviada por mim: SenderAlt é o MEU telefone, RecipientAlt é o da pessoa.
	info := types.MessageInfo{MessageSource: types.MessageSource{
		Chat: lid, IsFromMe: true,
		SenderAlt:    mustJID(t, "5511000000000@s.whatsapp.net"),
		RecipientAlt: mustJID(t, "558382341576@s.whatsapp.net"),
	}}
	if got := resolveChatPN(client, info); got != "558382341576@s.whatsapp.net" {
		t.Fatalf("enviada: got %q (não pode ser o telefone da conexão)", got)
	}
}

func TestResolveChatPN_UnknownReturnsEmpty(t *testing.T) {
	lid := mustJID(t, "999999999@lid")
	client := newClientWithLIDStore(t, &fakeLIDStore{})
	if got := resolveChatPN(client, types.MessageInfo{MessageSource: types.MessageSource{Chat: lid, Sender: lid}}); got != "" {
		t.Fatalf("sem mapa e sem alt deve ser vazio, veio %q", got)
	}
}

func TestResolveChatPN_NotLIDChatOrGroupIsEmpty(t *testing.T) {
	client := newClientWithLIDStore(t, &fakeLIDStore{})

	pnChat := types.MessageInfo{MessageSource: types.MessageSource{Chat: mustJID(t, "558382341576@s.whatsapp.net")}}
	if got := resolveChatPN(client, pnChat); got != "" {
		t.Fatalf("chat que já é telefone não precisa resolver, veio %q", got)
	}

	group := types.MessageInfo{MessageSource: types.MessageSource{Chat: mustJID(t, "120363000000000000@g.us"), IsGroup: true}}
	if got := resolveChatPN(client, group); got != "" {
		t.Fatalf("grupo não resolve, veio %q", got)
	}
}

func TestResolveChatPN_IgnoresNonPhoneAlt(t *testing.T) {
	lid := mustJID(t, "163423740493865@lid")
	client := newClientWithLIDStore(t, &fakeLIDStore{})
	info := types.MessageInfo{MessageSource: types.MessageSource{
		Chat: lid, Sender: lid, SenderAlt: mustJID(t, "55555555@lid"),
	}}
	if got := resolveChatPN(client, info); got != "" {
		t.Fatalf("alt que também é LID não é telefone, veio %q", got)
	}
}
