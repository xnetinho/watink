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

// Payload REAL capturado do aparelho (16:04): botão "Pagar com PIX" do WhatsApp Business.
const pixParamsReal = `{"currency":"BRL","total_amount":{"value":0,"offset":100},"reference_id":"4WK2I6WDX86","type":"physical-goods","order":{"status":"pending"},"payment_settings":[{"type":"pix_static_code","pix_static_code":{"merchant_name":"Meu PIX","key":"+5583988095596","key_type":"PHONE"}}],"share_payment_status":false,"referral":"chat_attachment"}`

func TestExtractMessageContent_PixReal(t *testing.T) {
	m := &waProto.Message{InteractiveMessage: &waProto.InteractiveMessage{InteractiveMessage: &waProto.InteractiveMessage_NativeFlowMessage_{
		NativeFlowMessage: &waProto.InteractiveMessage_NativeFlowMessage{Buttons: []*waProto.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			{Name: proto.String("payment_info"), ButtonParamsJSON: proto.String(pixParamsReal)}}}}}}
	got := extractMessageContent(m)
	if got.msgType != "pix" {
		t.Fatalf("type=%q, esperado pix (body=%q)", got.msgType, got.body)
	}
	if got.rich["key"] != "+5583988095596" || got.rich["keyType"] != "PHONE" || got.rich["name"] != "Meu PIX" {
		t.Fatalf("rich=%v", got.rich)
	}
	if _, has := got.rich["amount"]; has {
		t.Fatal("valor 0 não é cobrança: não deve aparecer")
	}
}

func TestExtractMessageContent_EventoEstruturado(t *testing.T) {
	ev := &waProto.EventMessage{Name: proto.String("Reunião"), JoinLink: proto.String("https://call.whatsapp.com/video/abc"), StartTime: proto.Int64(1793116800), IsScheduleCall: proto.Bool(true)}
	got := extractMessageContent(&waProto.Message{EventMessage: ev})
	if got.rich["joinLink"] != "https://call.whatsapp.com/video/abc" || got.rich["start"] != int64(1793116800) || got.rich["call"] != true {
		t.Fatalf("rich=%v", got.rich)
	}
	ev.JoinLink = proto.String("javascript:alert(1)")
	if _, has := extractMessageContent(&waProto.Message{EventMessage: ev}).rich["joinLink"]; has {
		t.Fatal("link que não é https nunca vai para o chat")
	}
}

func TestExtractMessageContent_EnqueteEstruturada(t *testing.T) {
	p := &waProto.PollCreationMessage{Name: proto.String("Q?"), SelectableOptionsCount: proto.Uint32(2), Options: []*waProto.PollCreationMessage_Option{{OptionName: proto.String("A")}, {OptionName: proto.String("B")}}}
	got := extractMessageContent(&waProto.Message{PollCreationMessageV3: p})
	if got.rich["selectable"] != 2 || len(got.rich["options"].([]string)) != 2 {
		t.Fatalf("rich=%v", got.rich)
	}
	p.SelectableOptionsCount = proto.Uint32(1)
	if extractMessageContent(&waProto.Message{PollCreationMessageV3: p}).rich["selectable"] != 1 {
		t.Fatal("contagem 1 = escolha única")
	}
}

func TestForwardedOf(t *testing.T) {
	img := &waProto.Message{ImageMessage: &waProto.ImageMessage{ContextInfo: &waProto.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(127)}}}
	if f, s := forwardedOf(img); !f || s != 127 {
		t.Fatalf("imagem: %v %d", f, s)
	}
	txt := &waProto.Message{ExtendedTextMessage: &waProto.ExtendedTextMessage{Text: proto.String("x"), ContextInfo: &waProto.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(1)}}}
	if f, s := forwardedOf(txt); !f || s != 1 {
		t.Fatalf("texto: %v %d", f, s)
	}
	wrapped := &waProto.Message{EphemeralMessage: &waProto.FutureProofMessage{Message: img}}
	if f, _ := forwardedOf(wrapped); !f {
		t.Fatal("encaminhada dentro de envelope temporário")
	}
	if f, _ := forwardedOf(&waProto.Message{Conversation: proto.String("oi")}); f {
		t.Fatal("mensagem comum não é encaminhada")
	}
}

// Enquete múltipla: o WhatsApp manda selectableOptionsCount 0 (sem limite). Antes virava "escolha 1".
func TestExtractMessageContent_EnqueteMultiplaSemLimite(t *testing.T) {
	p := &waProto.PollCreationMessage{Name: proto.String("Q?"), SelectableOptionsCount: proto.Uint32(0), Options: []*waProto.PollCreationMessage_Option{{OptionName: proto.String("A")}, {OptionName: proto.String("B")}}}
	if got := extractMessageContent(&waProto.Message{PollCreationMessageV3: p}).rich["selectable"]; got != 2 {
		t.Fatalf("selectable=%v, esperado 2 (todas)", got)
	}
}

// Recado de vídeo do WhatsApp (PTV) é um vídeo redondo, baixável como qualquer vídeo.
func TestExtractMessageContent_RecadoDeVideo(t *testing.T) {
	ptv := &waProto.VideoMessage{Mimetype: proto.String("video/mp4"), FileLength: proto.Uint64(877667)}
	got := extractMessageContent(&waProto.Message{PtvMessage: ptv})
	if got.msgType != "video" || got.rich["round"] != true || got.protoB64 == "" || got.fileLength != 877667 {
		t.Fatalf("%+v", got)
	}
}
