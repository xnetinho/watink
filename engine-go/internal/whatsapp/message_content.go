package whatsapp

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type mediaContent struct {
	body      string
	msgType   string
	mimeType  string
	thumbnail string // base64 JPEG preview (image/video/document); empty otherwise
	protoB64  string // base64 proto.Marshal of the downloadable media message
}

// unwrap abre os envelopes do WhatsApp (mensagem temporária, documento com legenda, visualização única, edição): o
// conteúdo real está dentro e deve ser classificado como se tivesse chegado direto.
func unwrap(msg *waProto.Message) *waProto.Message {
	for i := 0; i < 4 && msg != nil; i++ {
		var inner *waProto.Message
		switch {
		case msg.GetEphemeralMessage().GetMessage() != nil:
			inner = msg.GetEphemeralMessage().GetMessage()
		case msg.GetDocumentWithCaptionMessage().GetMessage() != nil:
			inner = msg.GetDocumentWithCaptionMessage().GetMessage()
		case msg.GetViewOnceMessageV2().GetMessage() != nil:
			inner = msg.GetViewOnceMessageV2().GetMessage()
		case msg.GetViewOnceMessage().GetMessage() != nil:
			inner = msg.GetViewOnceMessage().GetMessage()
		case msg.GetEditedMessage().GetMessage() != nil:
			inner = msg.GetEditedMessage().GetMessage()
		default:
			return msg
		}
		msg = inner
	}
	return msg
}

func extractMessageContent(msg *waProto.Message) mediaContent {
	msg = unwrap(msg)
	c := mediaContent{msgType: "chat", body: msg.GetConversation()}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		c.body = ext.GetText()
	}
	if img := msg.GetImageMessage(); img != nil {
		return mediaContent{body: img.GetCaption(), msgType: "image", mimeType: img.GetMimetype(), thumbnail: encodeThumb(img.GetJPEGThumbnail()), protoB64: marshalMedia(img)}
	}
	if video := msg.GetVideoMessage(); video != nil {
		return mediaContent{body: video.GetCaption(), msgType: "video", mimeType: video.GetMimetype(), thumbnail: encodeThumb(video.GetJPEGThumbnail()), protoB64: marshalMedia(video)}
	}
	if audio := msg.GetAudioMessage(); audio != nil {
		return mediaContent{msgType: "audio", mimeType: audio.GetMimetype(), protoB64: marshalMedia(audio)}
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		caption := doc.GetCaption()
		if caption == "" {
			caption = doc.GetTitle()
		}
		return mediaContent{body: caption, msgType: "document", mimeType: doc.GetMimetype(), thumbnail: encodeThumb(doc.GetJPEGThumbnail()), protoB64: marshalMedia(doc)}
	}
	if sticker := msg.GetStickerMessage(); sticker != nil {
		return mediaContent{msgType: "sticker", mimeType: sticker.GetMimetype(), protoB64: marshalMedia(sticker)}
	}
	if c.body != "" {
		return c
	}
	if loc := msg.GetLocationMessage(); loc != nil {
		return locationContent(loc.GetDegreesLatitude(), loc.GetDegreesLongitude(), loc.GetName(), loc.GetAddress(), loc.GetJPEGThumbnail())
	}
	if live := msg.GetLiveLocationMessage(); live != nil {
		return locationContent(live.GetDegreesLatitude(), live.GetDegreesLongitude(), "Localização em tempo real", live.GetCaption(), live.GetJPEGThumbnail())
	}
	if ct := msg.GetContactMessage(); ct != nil {
		return mediaContent{msgType: "vcard", body: vcardOf(ct.GetDisplayName(), ct.GetVcard())}
	}
	if arr := msg.GetContactsArrayMessage(); arr != nil && len(arr.GetContacts()) == 1 {
		ct := arr.GetContacts()[0]
		return mediaContent{msgType: "vcard", body: vcardOf(ct.GetDisplayName(), ct.GetVcard())}
	}
	if poll := pollOf(msg); poll != nil {
		lines := []string{"📊 " + poll.GetName()}
		for _, o := range poll.GetOptions() {
			lines = append(lines, "• "+o.GetOptionName())
		}
		return mediaContent{msgType: "poll", body: strings.Join(lines, "\n")}
	}
	if ev := msg.GetEventMessage(); ev != nil {
		body := "📅 " + ev.GetName()
		if loc := ev.GetLocation().GetName(); loc != "" {
			body += "\n📍 " + loc
		}
		return mediaContent{msgType: "event", body: body}
	}
	// Futuro-prova: qualquer conteúdo que o WhatsApp crie e nós não conheçamos vira um aviso, em vez de sumir.
	if kind := unsupportedKind(msg); kind != "" {
		return mediaContent{msgType: "unsupported", body: kind}
	}
	return c
}

func pollOf(msg *waProto.Message) *waProto.PollCreationMessage {
	for _, p := range []*waProto.PollCreationMessage{msg.GetPollCreationMessage(), msg.GetPollCreationMessageV2(), msg.GetPollCreationMessageV3()} {
		if p != nil {
			return p
		}
	}
	return nil
}

// locationContent usa o formato que o chat já lê: "imagem|link|descrição". A miniatura segue em thumbnail.
func locationContent(lat, lon float64, name, address string, thumb []byte) mediaContent {
	link := fmt.Sprintf("https://maps.google.com/maps?q=%f%%2C%f&z=17&hl=pt-BR", lat, lon)
	desc := strings.TrimSpace(strings.Join([]string{name, address}, "\n"))
	desc = strings.ReplaceAll(desc, "|", "/")
	return mediaContent{msgType: "location", body: "|" + link + "|" + desc, thumbnail: encodeThumb(thumb)}
}

func vcardOf(displayName, vcard string) string {
	if strings.TrimSpace(vcard) != "" {
		return vcard
	}
	return "BEGIN:VCARD\nVERSION:3.0\nFN:" + displayName + "\nEND:VCARD"
}

// handledOrTechnical lista o que extractMessageContent já trata ou que não é conteúdo para o operador (chaves de
// grupo, contexto, reações, edições...). Todo outro campo preenchido é um tipo que ainda não suportamos.
var handledOrTechnical = map[string]bool{
	"conversation": true, "extendedTextMessage": true, "imageMessage": true, "videoMessage": true, "audioMessage": true,
	"documentMessage": true, "stickerMessage": true, "locationMessage": true, "liveLocationMessage": true,
	"contactMessage": true, "contactsArrayMessage": true, "eventMessage": true,
	"pollCreationMessage": true, "pollCreationMessageV2": true, "pollCreationMessageV3": true,
	"senderKeyDistributionMessage": true, "fastRatchetKeySenderKeyDistributionMessage": true, "messageContextInfo": true,
	"protocolMessage": true, "reactionMessage": true, "encReactionMessage": true, "pollUpdateMessage": true,
	"keepInChatMessage": true, "pinInChatMessage": true, "deviceSentMessage": true, "placeholderMessage": true,
	"secretEncryptedMessage": true, "messageHistoryBundle": true, "messageHistoryNotice": true,
	"encEventResponseMessage": true, "albumMessage": true, "associatedChildMessage": true, "call": true,
	"callLogMesssage": true, "editedMessage": true,
	// envelopes: o conteúdo é desembrulhado por unwrap antes de classificar.
	"ephemeralMessage": true, "documentWithCaptionMessage": true, "viewOnceMessage": true, "viewOnceMessageV2": true,
	"viewOnceMessageV2Extension": true,
}

// unsupportedKind devolve o nome do campo (ex.: "listMessage") do primeiro conteúdo desconhecido, ou "".
func unsupportedKind(msg *waProto.Message) string {
	kind := ""
	msg.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if !handledOrTechnical[fd.JSONName()] {
			kind = fd.JSONName()
			return false
		}
		return true
	})
	return kind
}

// extractMentionedJIDs returns the @-mentioned JIDs carried in a message's
// ContextInfo, across every message type that can have mentions (a plain
// text reply is a Conversation with no ContextInfo — only ExtendedTextMessage
// and caption-bearing media types carry one). Used to decide whether an
// Assistant configured to only respond when mentioned in a group should
// reply or just observe.
func extractMentionedJIDs(msg *waProto.Message) []string {
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if jids := ext.GetContextInfo().GetMentionedJID(); len(jids) > 0 {
			return jids
		}
	}
	if img := msg.GetImageMessage(); img != nil {
		if jids := img.GetContextInfo().GetMentionedJID(); len(jids) > 0 {
			return jids
		}
	}
	if video := msg.GetVideoMessage(); video != nil {
		if jids := video.GetContextInfo().GetMentionedJID(); len(jids) > 0 {
			return jids
		}
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		if jids := doc.GetContextInfo().GetMentionedJID(); len(jids) > 0 {
			return jids
		}
	}
	return nil
}

// extractQuotedStanzaID returns the id of the message this one replies to,
// across every message type that can carry a ContextInfo (a plain
// Conversation cannot — mirrors extractMentionedJIDs above). Empty string
// when the message isn't a reply. Feeds Message.QuotedMsgID
// (business/internal/services/event_payloads.go MessagePayload.QuotedMsgId),
// which the business side already declares and maps but nothing populates
// today — receive_message.go's QuotedMsgID branch never fires.
func extractQuotedStanzaID(msg *waProto.Message) string {
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if id := ext.GetContextInfo().GetStanzaID(); id != "" {
			return id
		}
	}
	if img := msg.GetImageMessage(); img != nil {
		if id := img.GetContextInfo().GetStanzaID(); id != "" {
			return id
		}
	}
	if video := msg.GetVideoMessage(); video != nil {
		if id := video.GetContextInfo().GetStanzaID(); id != "" {
			return id
		}
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		if id := doc.GetContextInfo().GetStanzaID(); id != "" {
			return id
		}
	}
	if audio := msg.GetAudioMessage(); audio != nil {
		if id := audio.GetContextInfo().GetStanzaID(); id != "" {
			return id
		}
	}
	return ""
}

func encodeThumb(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

func marshalMedia(m proto.Message) string {
	raw, err := proto.Marshal(m)
	if err != nil {
		log.Printf("failed to marshal media proto: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(raw)
}
