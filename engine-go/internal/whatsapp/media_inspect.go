package whatsapp

import (
	"fmt"
	"os"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// inspectMedia descreve os CAMPOS de metadados de uma mídia recebida (nunca o conteúdo nem a chave): é como se enxerga o
// que o cliente oficial do WhatsApp envia e o nosso envio ainda não envia. Ligado por WATINK_MEDIA_INSPECT=1.
func inspectMedia(msg *waProto.Message) string {
	msg = unwrap(msg)
	if msg == nil {
		return ""
	}
	switch {
	case msg.GetAudioMessage() != nil:
		a := msg.GetAudioMessage()
		// ptt: GetPTT() devolve false tanto para "ausente" quanto para "false explícito", e é essa diferença que decide se
		// o celular desenha o áudio como mensagem de voz ou como arquivo. Por isso mostra se o campo EXISTE.
		ptt := "ausente"
		if a.PTT != nil {
			ptt = fmt.Sprintf("%v", *a.PTT)
		}
		return fmt.Sprintf("audio mime=%q seconds=%d ptt=%s waveform=%dB mediaKeyTs=%d streamingSidecar=%dB viewOnce=%v fileLen=%d",
			a.GetMimetype(), a.GetSeconds(), ptt, len(a.GetWaveform()), a.GetMediaKeyTimestamp(), len(a.GetStreamingSidecar()), a.GetViewOnce(), a.GetFileLength())
	case msg.GetDocumentMessage() != nil:
		d := msg.GetDocumentMessage()
		return fmt.Sprintf("document mime=%q title=%q fileName=%q pageCount=%d thumb=%dB thumbWH=%dx%d thumbDirectPath=%v thumbSHA=%dB thumbEncSHA=%dB mediaKeyTs=%d contactVcard=%v fileLen=%d",
			d.GetMimetype(), d.GetTitle(), d.GetFileName(), d.GetPageCount(), len(d.GetJPEGThumbnail()), d.GetThumbnailWidth(), d.GetThumbnailHeight(),
			d.GetThumbnailDirectPath() != "", len(d.GetThumbnailSHA256()), len(d.GetThumbnailEncSHA256()), d.GetMediaKeyTimestamp(), d.GetContactVcard(), d.GetFileLength())
	case msg.GetImageMessage() != nil:
		i := msg.GetImageMessage()
		return fmt.Sprintf("image mime=%q wh=%dx%d thumb=%dB thumbDirectPath=%v firstScanSidecar=%dB midQualityFileSHA=%dB mediaKeyTs=%d fileLen=%d",
			i.GetMimetype(), i.GetWidth(), i.GetHeight(), len(i.GetJPEGThumbnail()), i.GetThumbnailDirectPath() != "", len(i.GetFirstScanSidecar()), len(i.GetMidQualityFileSHA256()), i.GetMediaKeyTimestamp(), i.GetFileLength())
	case msg.GetVideoMessage() != nil:
		v := msg.GetVideoMessage()
		return fmt.Sprintf("video mime=%q seconds=%d wh=%dx%d thumb=%dB thumbDirectPath=%v streamingSidecar=%dB gif=%v mediaKeyTs=%d fileLen=%d",
			v.GetMimetype(), v.GetSeconds(), v.GetWidth(), v.GetHeight(), len(v.GetJPEGThumbnail()), v.GetThumbnailDirectPath() != "", len(v.GetStreamingSidecar()), v.GetGifPlayback(), v.GetMediaKeyTimestamp(), v.GetFileLength())
	}
	return ""
}

// inspectQuote descreve o que o cliente oficial põe em ContextInfo.QuotedMessage ao responder (tipo e campos presentes,
// nunca chaves nem conteúdo), para o nosso envio citar do mesmo jeito.
func inspectQuote(msg *waProto.Message) string {
	msg = unwrap(msg)
	var ci *waProto.ContextInfo
	switch {
	case msg.GetExtendedTextMessage() != nil:
		ci = msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		ci = msg.GetAudioMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		ci = msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		ci = msg.GetVideoMessage().GetContextInfo()
	case msg.GetDocumentMessage() != nil:
		ci = msg.GetDocumentMessage().GetContextInfo()
	}
	if ci == nil || ci.GetStanzaID() == "" {
		return ""
	}
	q := ci.GetQuotedMessage()
	desc := "vazia"
	switch {
	case q == nil:
	case q.GetAudioMessage() != nil:
		a := q.GetAudioMessage()
		ptt := "ausente"
		if a.PTT != nil {
			ptt = fmt.Sprintf("%v", *a.PTT)
		}
		desc = fmt.Sprintf("audio mime=%q seconds=%d ptt=%s waveform=%dB url=%v directPath=%v mediaKey=%dB sha=%dB fileLen=%d", a.GetMimetype(), a.GetSeconds(), ptt, len(a.GetWaveform()), a.GetURL() != "", a.GetDirectPath() != "", len(a.GetMediaKey()), len(a.GetFileSHA256()), a.GetFileLength())
	case q.GetImageMessage() != nil:
		i := q.GetImageMessage()
		desc = fmt.Sprintf("image mime=%q caption=%q wh=%dx%d thumb=%dB url=%v mediaKey=%dB", i.GetMimetype(), i.GetCaption(), i.GetWidth(), i.GetHeight(), len(i.GetJPEGThumbnail()), i.GetURL() != "", len(i.GetMediaKey()))
	case q.GetVideoMessage() != nil:
		v := q.GetVideoMessage()
		desc = fmt.Sprintf("video mime=%q caption=%q seconds=%d thumb=%dB url=%v mediaKey=%dB", v.GetMimetype(), v.GetCaption(), v.GetSeconds(), len(v.GetJPEGThumbnail()), v.GetURL() != "", len(v.GetMediaKey()))
	case q.GetDocumentMessage() != nil:
		d := q.GetDocumentMessage()
		desc = fmt.Sprintf("document mime=%q fileName=%q title=%q pages=%d thumb=%dB url=%v", d.GetMimetype(), d.GetFileName(), d.GetTitle(), d.GetPageCount(), len(d.GetJPEGThumbnail()), d.GetURL() != "")
	case q.GetConversation() != "":
		desc = "conversation"
	case q.GetExtendedTextMessage() != nil:
		desc = "extendedText"
	default:
		desc = "outro"
	}
	return fmt.Sprintf("stanza=%s participant=%q remoteJid=%q quoted=%s", ci.GetStanzaID(), ci.GetParticipant(), ci.GetRemoteJID(), desc)
}

func mediaInspectEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("WATINK_MEDIA_INSPECT")))
	return v == "1" || v == "true"
}
