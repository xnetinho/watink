package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/util/gcmutil"
	"go.mau.fi/whatsmeow/util/hkdfutil"
	"google.golang.org/protobuf/proto"
)

var editChat = types.NewJID("5511999990001", types.DefaultUserServer)

// memSecrets guarda o segredo das mensagens em memória (o whatsmeow o guarda no banco ao receber cada mensagem).
type memSecrets struct {
	secret []byte
	sender types.JID
}

func (m *memSecrets) PutMessageSecrets(context.Context, []store.MessageSecretInsert) error {
	return nil
}
func (m *memSecrets) PutMessageSecret(context.Context, types.JID, types.JID, types.MessageID, []byte) error {
	return nil
}
func (m *memSecrets) GetMessageSecret(context.Context, types.JID, types.JID, types.MessageID) ([]byte, types.JID, error) {
	return m.secret, m.sender, nil
}

// secretEditEvent monta a edição NO FORMATO REAL: o celular do contato manda um SecretEncryptedMessage MESSAGE_EDIT
// (o texto novo cifrado com o segredo da mensagem original), e não o ProtocolMessage que o BuildEdit usa para enviar.
// Confirmado com edições reais no celular (o diagnóstico mostrou secretEncType=MESSAGE_EDIT e isEdit=false).
func secretEditEvent(t *testing.T, newContent *waProto.Message, original string) (*whatsmeow.Client, *events.Message) {
	t.Helper()
	// Formato REAL, capturado no celular: o conteúdo decifrado é um ProtocolMessage MESSAGE_EDIT com o texto novo em
	// editedMessage (e o timestampMS da edição). O texto NÃO vem na raiz.
	newContent = &waProto.Message{ProtocolMessage: &waProto.ProtocolMessage{
		Key:           &waCommon.MessageKey{RemoteJID: proto.String("202530038505654@lid"), FromMe: proto.Bool(true), ID: proto.String(original)},
		Type:          waProto.ProtocolMessage_MESSAGE_EDIT.Enum(),
		EditedMessage: newContent,
		TimestampMS:   proto.Int64(time.Date(2026, 10, 29, 12, 7, 0, 0, time.UTC).UnixMilli()),
	}}
	sender := types.NewJID("165923277262973", types.HiddenUserServer)
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i + 1)
	}

	plaintext, err := proto.Marshal(newContent)
	if err != nil {
		t.Fatal(err)
	}
	// mesma derivação do whatsmeow (generateMsgSecretKey, tipo "Message Edit"; sem additionalData para edição)
	info := append(append(append([]byte(original), sender.ToNonAD().String()...), sender.ToNonAD().String()...), "Message Edit"...)
	key := hkdfutil.SHA256(secret, nil, info, 32)
	iv := make([]byte, 12)
	for i := range iv {
		iv[i] = byte(100 + i)
	}
	payload, err := gcmutil.Encrypt(key, iv, plaintext, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &whatsmeow.Client{Store: &store.Device{MsgSecrets: &memSecrets{secret: secret, sender: sender}}}
	evt := &events.Message{
		Info: types.MessageInfo{
			ID:            "ID-DO-EVENTO-DE-EDICAO",
			Timestamp:     time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC),
			MessageSource: types.MessageSource{Chat: sender, Sender: sender},
		},
		Message: &waProto.Message{SecretEncryptedMessage: &waE2E.SecretEncryptedMessage{
			TargetMessageKey: &waCommon.MessageKey{RemoteJID: proto.String(sender.String()), FromMe: proto.Bool(true), ID: proto.String(original)},
			EncPayload:       payload,
			EncIV:            iv,
			SecretEncType:    waE2E.SecretEncryptedMessage_MESSAGE_EDIT.Enum(),
		}},
	}
	return client, evt
}

func TestEdicaoRecebida_FormatoRealDoCelularPublicaMessageEdited(t *testing.T) {
	svc, calls := newTestService()
	client, evt := secretEditEvent(t, &waProto.Message{Conversation: proto.String("texto NOVO")}, "ORIGINAL123")

	svc.handleMessageEvent(client, 1, "tenant-x", evt)

	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento, veio %d (antes a edição era descartada em silêncio)", len(*calls))
	}
	c := (*calls)[0]
	if c.eventType != "message.edited" {
		t.Fatalf("evento = %q, esperava message.edited", c.eventType)
	}
	if c.payload["messageId"] != "ORIGINAL123" {
		t.Errorf("messageId = %v: tem de ser o da mensagem ORIGINAL (targetMessageKey), não o do evento de edição", c.payload["messageId"])
	}
	if c.payload["body"] != "texto NOVO" {
		t.Errorf("body = %v", c.payload["body"])
	}
	if c.payload["editedAt"] != "2026-10-29T12:07:00Z" {
		t.Errorf("editedAt = %v: deve ser o timestampMS da edição, não a hora do evento", c.payload["editedAt"])
	}
}

func TestEdicaoRecebida_LegendaDeMidiaEOTexto(t *testing.T) {
	cases := map[string]*waProto.Message{
		"imagem":                     {ImageMessage: &waProto.ImageMessage{Caption: proto.String("legenda nova")}},
		"video":                      {VideoMessage: &waProto.VideoMessage{Caption: proto.String("legenda nova")}},
		"documento":                  {DocumentMessage: &waProto.DocumentMessage{Caption: proto.String("legenda nova")}},
		"texto longo (ExtendedText)": {ExtendedTextMessage: &waProto.ExtendedTextMessage{Text: proto.String("legenda nova")}},
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			svc, calls := newTestService()
			client, evt := secretEditEvent(t, content, "ORIG")
			svc.handleMessageEvent(client, 1, "t", evt)
			if len(*calls) != 1 || (*calls)[0].payload["body"] != "legenda nova" {
				t.Fatalf("esperava body=legenda nova, veio %+v", *calls)
			}
		})
	}
}

// Sem o segredo da original (mensagem anterior à conexão) não há como ler o texto novo: não publica nada, e nunca
// transforma a edição numa mensagem nova.
func TestEdicaoRecebida_SemOSegredoDaOriginalNaoPublicaNada(t *testing.T) {
	svc, calls := newTestService()
	client, evt := secretEditEvent(t, &waProto.Message{Conversation: proto.String("x")}, "ORIG")
	client.Store.MsgSecrets = &memSecrets{secret: nil}

	svc.handleMessageEvent(client, 1, "t", evt)

	if len(*calls) != 0 {
		t.Fatalf("sem o segredo não há o que publicar, veio %+v", *calls)
	}
}

// O formato de envio (ProtocolMessage via BuildEdit) também é tratado: é o que o próprio celular pareado manda quando EU
// edito por ele, e o formato antigo de alguns clientes.
func TestEdicaoRecebida_FormatoProtocolMessage(t *testing.T) {
	svc, calls := newTestService()
	raw := (&whatsmeow.Client{}).BuildEdit(editChat, "ORIGINAL123", &waProto.Message{Conversation: proto.String("texto NOVO")})
	evt := (&events.Message{
		Info:       types.MessageInfo{ID: "EVT", Timestamp: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC), MessageSource: types.MessageSource{Chat: editChat, Sender: editChat}},
		RawMessage: raw,
	}).UnwrapRaw()

	svc.handleMessageEvent(nil, 1, "t", evt)

	if len(*calls) != 1 || (*calls)[0].eventType != "message.edited" || (*calls)[0].payload["messageId"] != "ORIGINAL123" {
		t.Fatalf("veio %+v", *calls)
	}
}

func TestEdicaoRecebida_InstanteDoProtocolMessageVemDoAparelho(t *testing.T) {
	raw := (&whatsmeow.Client{}).BuildEdit(editChat, "O1", &waProto.Message{Conversation: proto.String("x")})
	evt := (&events.Message{Info: types.MessageInfo{Timestamp: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}, RawMessage: raw}).UnwrapRaw()
	pm := evt.Message.GetProtocolMessage()
	pm.TimestampMS = proto.Int64(time.Date(2026, 10, 29, 12, 5, 0, 0, time.UTC).UnixMilli())

	if p := editedPayload(1, evt, "O1", pm); p["editedAt"] != "2026-10-29T12:05:00Z" {
		t.Errorf("editedAt = %v, esperava o TimestampMS da edição", p["editedAt"])
	}
	pm.TimestampMS = nil
	if p := editedPayload(1, evt, "O1", pm); p["editedAt"] != "2026-10-29T12:00:00Z" {
		t.Errorf("sem TimestampMS cai no horário da mensagem, veio %v", p["editedAt"])
	}
}
