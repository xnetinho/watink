package whatsapp

import (
	"strings"
	"testing"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

func TestExtractMessageContent_TiposQueChegavamEmBranco(t *testing.T) {
	loc := &waProto.Message{LocationMessage: &waProto.LocationMessage{DegreesLatitude: proto.Float64(-7.1), DegreesLongitude: proto.Float64(-34.8), Name: proto.String("Praça"), Address: proto.String("Centro")}}
	live := &waProto.Message{LiveLocationMessage: &waProto.LiveLocationMessage{DegreesLatitude: proto.Float64(-7.1), DegreesLongitude: proto.Float64(-34.8)}}
	card := &waProto.Message{ContactMessage: &waProto.ContactMessage{DisplayName: proto.String("Ana"), Vcard: proto.String("BEGIN:VCARD\nFN:Ana\nEND:VCARD")}}
	poll := &waProto.Message{PollCreationMessageV3: &waProto.PollCreationMessage{Name: proto.String("Almoço?"), Options: []*waProto.PollCreationMessage_Option{{OptionName: proto.String("Sim")}, {OptionName: proto.String("Não")}}}}
	event := &waProto.Message{EventMessage: &waProto.EventMessage{Name: proto.String("Reunião")}}

	cases := []struct {
		name, typ, contains string
		m                   *waProto.Message
	}{
		{"localização", "location", "maps.google.com/maps?q=-7.1", loc},
		{"localização ao vivo", "location", "maps.google.com", live},
		{"contato", "vcard", "FN:Ana", card},
		{"enquete", "poll", "Almoço?", poll},
		{"evento", "event", "Reunião", event},
	}
	for _, c := range cases {
		got := extractMessageContent(c.m)
		if got.msgType != c.typ || !strings.Contains(got.body, c.contains) {
			t.Errorf("%s: type=%q body=%q (esperado %q contendo %q)", c.name, got.msgType, got.body, c.typ, c.contains)
		}
	}
}

func TestExtractMessageContent_LocalizacaoNaoQuebraOFormatoDoChat(t *testing.T) {
	m := &waProto.Message{LocationMessage: &waProto.LocationMessage{DegreesLatitude: proto.Float64(1), DegreesLongitude: proto.Float64(2), Name: proto.String("a|b")}}
	if n := len(strings.Split(extractMessageContent(m).body, "|")); n != 3 {
		t.Fatalf("o chat divide o corpo por '|' em 3 partes (imagem|link|descrição), veio %d", n)
	}
}

// O WhatsApp cria tipos novos com o tempo: o que não conhecemos vira aviso, nunca some.
func TestExtractMessageContent_TipoDesconhecidoViraAviso(t *testing.T) {
	got := extractMessageContent(&waProto.Message{ListMessage: &waProto.ListMessage{}})
	if got.msgType != "unsupported" || got.body != "listMessage" {
		t.Fatalf("type=%q body=%q", got.msgType, got.body)
	}
}

func TestExtractMessageContent_TecnicoNaoViraAviso(t *testing.T) {
	for name, m := range map[string]*waProto.Message{
		"chave de grupo": {SenderKeyDistributionMessage: &waProto.SenderKeyDistributionMessage{}},
		"vazia":          {},
	} {
		if got := extractMessageContent(m); got.msgType == "unsupported" {
			t.Errorf("%s não é conteúdo para o operador, veio aviso (%q)", name, got.body)
		}
	}
}

// Chat com mensagens temporárias embrulha TODO conteúdo: texto nele não pode virar "não suportado".
func TestExtractMessageContent_DesembrulhaEnvelopes(t *testing.T) {
	txt := &waProto.Message{Conversation: proto.String("oi temporário")}
	got := extractMessageContent(&waProto.Message{EphemeralMessage: &waProto.FutureProofMessage{Message: txt}})
	if got.msgType != "chat" || got.body != "oi temporário" {
		t.Fatalf("temporária: type=%q body=%q", got.msgType, got.body)
	}
	doc := &waProto.Message{DocumentMessage: &waProto.DocumentMessage{Caption: proto.String("contrato"), Mimetype: proto.String("application/pdf")}}
	got = extractMessageContent(&waProto.Message{DocumentWithCaptionMessage: &waProto.FutureProofMessage{Message: doc}})
	if got.msgType != "document" || got.body != "contrato" {
		t.Fatalf("documento com legenda: type=%q body=%q", got.msgType, got.body)
	}
}
