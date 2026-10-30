package whatsapp

import (
	"encoding/base64"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
	"go.mau.fi/whatsmeow"
)

func TestBuildTextMessage_PlainConversation(t *testing.T) {
	msg := buildTextMessage("hello", "", "", "", nil, nil, forwardMark{})
	if msg.GetConversation() != "hello" {
		t.Fatalf("expected plain conversation, got %q", msg.GetConversation())
	}
	if msg.GetExtendedTextMessage() != nil {
		t.Fatal("plain message should not use ExtendedTextMessage")
	}
}

func TestBuildTextMessage_WithQuote(t *testing.T) {
	msg := buildTextMessage("reply", "QUOTED1", "5511@s.whatsapp.net", "", nil, nil, forwardMark{})
	ext := msg.GetExtendedTextMessage()
	if ext == nil {
		t.Fatal("quoted message should use ExtendedTextMessage")
	}
	if ext.GetText() != "reply" {
		t.Errorf("text = %q", ext.GetText())
	}
	if ext.GetContextInfo().GetStanzaID() != "QUOTED1" {
		t.Errorf("stanzaID = %q", ext.GetContextInfo().GetStanzaID())
	}
}

func TestBuildTextMessage_WithMentions(t *testing.T) {
	mentions := []string{"5511@s.whatsapp.net", "5522@s.whatsapp.net"}
	msg := buildTextMessage("hi @a @b", "", "", "", nil, mentions, forwardMark{})
	ext := msg.GetExtendedTextMessage()
	if ext == nil {
		t.Fatal("mention message should use ExtendedTextMessage")
	}
	if len(ext.GetContextInfo().GetMentionedJID()) != 2 {
		t.Errorf("expected 2 mentions, got %d", len(ext.GetContextInfo().GetMentionedJID()))
	}
}

func TestNormalizeMediaType(t *testing.T) {
	cases := map[string]whatsmeow.MediaType{
		"image":    whatsmeow.MediaImage,
		"IMAGE":    whatsmeow.MediaImage,
		"video":    whatsmeow.MediaVideo,
		"audio":    whatsmeow.MediaAudio,
		"sticker":  whatsmeow.MediaImage,
		"document": whatsmeow.MediaDocument,
		"weird":    whatsmeow.MediaDocument,
	}
	for in, want := range cases {
		if got := normalizeMediaType(in); got != want {
			t.Errorf("normalizeMediaType(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveMediaBytes_FromBase64(t *testing.T) {
	raw := []byte("binary-content")
	payload := MediaCommandPayload{MediaData: base64.StdEncoding.EncodeToString(raw)}
	got, err := resolveMediaBytes(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("got %q, want %q", got, raw)
	}
}

func TestResolveMediaBytes_MissingSource(t *testing.T) {
	_, err := resolveMediaBytes(MediaCommandPayload{})
	if err == nil {
		t.Fatal("expected error when neither mediaData nor mediaUrl is set")
	}
}

func TestBuildMediaMessage_Image(t *testing.T) {
	payload := MediaCommandPayload{MediaType: "image", Body: "caption", MimeType: "image/png"}
	msg := buildMediaMessage(payload, whatsmeow.UploadResponse{}, mediameta.Meta{})
	if msg.GetImageMessage() == nil {
		t.Fatal("expected ImageMessage")
	}
	if msg.GetImageMessage().GetCaption() != "caption" {
		t.Errorf("caption = %q", msg.GetImageMessage().GetCaption())
	}
}

func TestBuildMediaMessage_DocumentDefault(t *testing.T) {
	payload := MediaCommandPayload{MediaType: "application/zip", FileName: "f.zip"}
	msg := buildMediaMessage(payload, whatsmeow.UploadResponse{}, mediameta.Meta{})
	if msg.GetDocumentMessage() == nil {
		t.Fatal("expected DocumentMessage for non-media type")
	}
	if msg.GetDocumentMessage().GetFileName() != "f.zip" {
		t.Errorf("fileName = %q", msg.GetDocumentMessage().GetFileName())
	}
}

// O celular desenha a caixa de citação a partir do CONTEÚDO da mensagem citada (ContextInfo.QuotedMessage).
// Só com o StanzaID a resposta chegava sem a citação visível para o destinatário.
func TestBuildTextMessage_QuoteCarriesTheQuotedContent(t *testing.T) {
	msg := buildTextMessage("5 dias", "Q1", "5511@s.whatsapp.net", "qual o prazo?", nil, nil, forwardMark{})
	ci := msg.GetExtendedTextMessage().GetContextInfo()
	if ci.GetStanzaID() != "Q1" {
		t.Fatalf("stanzaID = %q", ci.GetStanzaID())
	}
	if got := ci.GetQuotedMessage().GetConversation(); got != "qual o prazo?" {
		t.Fatalf("QuotedMessage = %q, esperado o texto citado", got)
	}
}

func TestBuildTextMessage_QuoteWithoutBodyStillQuotesByID(t *testing.T) {
	ci := buildTextMessage("ok", "Q1", "", "", nil, nil, forwardMark{}).GetExtendedTextMessage().GetContextInfo()
	if ci.GetStanzaID() != "Q1" || ci.GetQuotedMessage() != nil {
		t.Fatalf("sem corpo citado deve citar só pelo id: %+v", ci)
	}
}

func TestBuildMediaMessage_QuoteCarriesTheQuotedContent(t *testing.T) {
	for _, mt := range []string{"image", "video", "audio", "document"} {
		msg := buildMediaMessage(MediaCommandPayload{MediaType: mt, QuotedMsgID: "Q1", QuotedJID: "5511@s.whatsapp.net", QuotedBody: "oi"}, whatsmeow.UploadResponse{}, mediameta.Meta{})
		var got string
		switch mt {
		case "image":
			got = msg.GetImageMessage().GetContextInfo().GetQuotedMessage().GetConversation()
		case "video":
			got = msg.GetVideoMessage().GetContextInfo().GetQuotedMessage().GetConversation()
		case "audio":
			got = msg.GetAudioMessage().GetContextInfo().GetQuotedMessage().GetConversation()
		default:
			got = msg.GetDocumentMessage().GetContextInfo().GetQuotedMessage().GetConversation()
		}
		if got != "oi" {
			t.Errorf("%s: QuotedMessage = %q, esperado %q", mt, got, "oi")
		}
	}
}

func TestBuildTextMessage_EncaminhadaLevaSeloEScore(t *testing.T) {
	ci := buildTextMessage("oi", "", "", "", nil, nil, forwardInfo(true, 3)).GetExtendedTextMessage().GetContextInfo()
	if !ci.GetIsForwarded() || ci.GetForwardingScore() != 3 {
		t.Fatalf("selo/score ausentes: %v", ci)
	}
	if plain := buildTextMessage("oi", "", "", "", nil, nil, forwardInfo(false, 3)); plain.GetConversation() != "oi" {
		t.Fatal("sem encaminhar, o texto continua simples")
	}
	if ci := buildTextMessage("oi", "", "", "", nil, nil, forwardInfo(true, 0)).GetExtendedTextMessage().GetContextInfo(); ci.GetForwardingScore() != 1 {
		t.Fatal("score mínimo é 1")
	}
}

func TestBuildMediaMessage_EncaminhadaESticker(t *testing.T) {
	for _, mt := range []string{"image", "video", "audio", "document", "sticker"} {
		m := buildMediaMessage(MediaCommandPayload{MediaType: mt, Forwarded: true, ForwardScore: 2}, whatsmeow.UploadResponse{}, mediameta.Meta{})
		var ci interface{ GetIsForwarded() bool }
		switch mt {
		case "image":
			ci = m.GetImageMessage().GetContextInfo()
		case "video":
			ci = m.GetVideoMessage().GetContextInfo()
		case "audio":
			ci = m.GetAudioMessage().GetContextInfo()
		case "sticker":
			ci = m.GetStickerMessage().GetContextInfo()
		default:
			ci = m.GetDocumentMessage().GetContextInfo()
		}
		if !ci.GetIsForwarded() {
			t.Errorf("%s: sem selo de encaminhada", mt)
		}
	}
}

func TestBuildMediaMessage_ImagemEVideoLevamMiniaturaEDimensoes(t *testing.T) {
	meta := mediameta.Meta{Width: 640, Height: 480, Seconds: 7, Thumbnail: []byte{0xff, 0xd8, 0xff}}

	img := buildMediaMessage(MediaCommandPayload{MediaType: "image", MimeType: "image/png"}, whatsmeow.UploadResponse{}, meta).GetImageMessage()
	if img.GetWidth() != 640 || img.GetHeight() != 480 || len(img.GetJPEGThumbnail()) != 3 || img.GetMediaKeyTimestamp() == 0 {
		t.Fatalf("imagem sem metadados: w=%d h=%d thumb=%d ts=%d", img.GetWidth(), img.GetHeight(), len(img.GetJPEGThumbnail()), img.GetMediaKeyTimestamp())
	}

	vid := buildMediaMessage(MediaCommandPayload{MediaType: "video", MimeType: "video/mp4"}, whatsmeow.UploadResponse{}, meta).GetVideoMessage()
	if vid.GetSeconds() != 7 || vid.GetWidth() != 640 || len(vid.GetJPEGThumbnail()) != 3 {
		t.Fatalf("vídeo sem metadados: s=%d w=%d thumb=%d", vid.GetSeconds(), vid.GetWidth(), len(vid.GetJPEGThumbnail()))
	}
}

func TestBuildMediaMessage_SemMetadadosOsCamposFicamAusentes(t *testing.T) {
	img := buildMediaMessage(MediaCommandPayload{MediaType: "image"}, whatsmeow.UploadResponse{}, mediameta.Meta{}).GetImageMessage()
	if img.Width != nil || img.Height != nil || img.JPEGThumbnail != nil {
		t.Fatal("sem ffmpeg os campos devem ficar ausentes (nunca 0), como o cliente oficial quando não sabe o valor")
	}
}

func TestBuildMediaMessage_GravacaoEhPTTEAmbosLevamOnda(t *testing.T) {
	meta := mediameta.Meta{Seconds: 5, Waveform: []byte{1, 2, 3}}

	voice := buildMediaMessage(MediaCommandPayload{MediaType: "voice", MimeType: "audio/ogg; codecs=opus"}, whatsmeow.UploadResponse{}, meta).GetAudioMessage()
	if !voice.GetPTT() || len(voice.GetWaveform()) != 3 || voice.GetSeconds() != 5 {
		t.Fatalf("gravação: ptt=%v onda=%d s=%d", voice.GetPTT(), len(voice.GetWaveform()), voice.GetSeconds())
	}

	file := buildMediaMessage(MediaCommandPayload{MediaType: "audio", MimeType: "audio/mpeg"}, whatsmeow.UploadResponse{}, meta).GetAudioMessage()
	if file.GetPTT() {
		t.Fatal("arquivo de áudio não pode ser PTT")
	}
	if file.PTT == nil {
		t.Fatal("PTT precisa ser EXPLÍCITO (false) no arquivo de áudio: ausente faz o celular desenhar como mensagem de voz")
	}
	if file.GetSeconds() != 5 {
		t.Fatalf("arquivo também mostra a duração, veio %d", file.GetSeconds())
	}
}

func TestNormalizeMediaType_VoiceSobeComoAudio(t *testing.T) {
	if normalizeMediaType("voice") != whatsmeow.MediaAudio || normalizeMediaType("VOICE") != whatsmeow.MediaAudio {
		t.Fatal("voice precisa subir como áudio")
	}
}

func TestBuildMediaMessage_PDFLevaMiniaturaEPaginas(t *testing.T) {
	meta := mediameta.Meta{Pages: 6, Thumbnail: []byte{0xff, 0xd8, 0xff, 1}, ThumbW: 51, ThumbH: 72}
	doc := buildMediaMessage(MediaCommandPayload{MediaType: "document", MimeType: "application/pdf", FileName: "a.pdf"}, whatsmeow.UploadResponse{}, meta).GetDocumentMessage()
	if doc.GetPageCount() != 6 || len(doc.GetJPEGThumbnail()) != 4 || doc.GetThumbnailWidth() != 51 || doc.GetThumbnailHeight() != 72 {
		t.Fatalf("pdf: páginas=%d miniatura=%dB %dx%d", doc.GetPageCount(), len(doc.GetJPEGThumbnail()), doc.GetThumbnailWidth(), doc.GetThumbnailHeight())
	}
	plain := buildMediaMessage(MediaCommandPayload{MediaType: "document", MimeType: "application/zip", FileName: "a.zip"}, whatsmeow.UploadResponse{}, mediameta.Meta{}).GetDocumentMessage()
	if plain.PageCount != nil || plain.JPEGThumbnail != nil || plain.ThumbnailWidth != nil {
		t.Fatal("documento sem metadados deve deixar os campos ausentes")
	}
}

func TestNormalizeAudioMime(t *testing.T) {
	for in, want := range map[string]string{
		"audio/ogg":               "audio/ogg; codecs=opus",
		"AUDIO/OGG":               "audio/ogg; codecs=opus",
		"application/ogg":         "audio/ogg; codecs=opus",
		"audio/ogg; codecs=opus":  "audio/ogg; codecs=opus",
		"audio/ogg;codecs=vorbis": "audio/ogg;codecs=vorbis",
		"audio/mpeg":              "audio/mpeg",
		"audio/mp4":               "audio/mp4",
		"":                        "",
	} {
		if got := normalizeAudioMime(in); got != want {
			t.Errorf("normalizeAudioMime(%q) = %q, esperado %q", in, got, want)
		}
	}
}

// Citar um áudio: o cliente oficial manda um AudioMessage (voz, duração) dentro da citação, não o texto do rótulo.
func TestBuildTextMessage_QuotedVoiceIsAnAudioMessage(t *testing.T) {
	qm := &QuotedMedia{Type: "audio", MimeType: "audio/ogg", Voice: true, Seconds: 4}
	ci := buildTextMessage("ok", "Q1", "5511@lid", "🎵 Áudio", qm, nil, forwardMark{}).GetExtendedTextMessage().GetContextInfo()
	a := ci.GetQuotedMessage().GetAudioMessage()
	if a == nil {
		t.Fatalf("a citação devia ser um AudioMessage, veio %+v", ci.GetQuotedMessage())
	}
	if !a.GetPTT() || a.GetSeconds() != 4 || a.GetMimetype() != "audio/ogg; codecs=opus" {
		t.Fatalf("áudio citado = ptt %v, %ds, %q", a.GetPTT(), a.GetSeconds(), a.GetMimetype())
	}
	file := buildTextMessage("ok", "Q1", "5511@lid", "🎵 Áudio", &QuotedMedia{Type: "audio", MimeType: "audio/mpeg"}, nil, forwardMark{})
	if p := file.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage().GetAudioMessage(); p == nil || p.PTT == nil || p.GetPTT() {
		t.Fatal("arquivo de áudio citado deve levar PTT=false explícito")
	}
}

func TestBuildTextMessage_QuotedImageCarriesThumbAndCaption(t *testing.T) {
	qm := &QuotedMedia{Type: "image", MimeType: "image/jpeg", Caption: "olha", Thumbnail: "QUJD"}
	img := buildTextMessage("ok", "Q1", "", "", qm, nil, forwardMark{}).GetExtendedTextMessage().GetContextInfo().GetQuotedMessage().GetImageMessage()
	if img.GetCaption() != "olha" || string(img.GetJPEGThumbnail()) != "ABC" {
		t.Fatalf("imagem citada = %q %q", img.GetCaption(), img.GetJPEGThumbnail())
	}
}

// O cliente oficial não preenche remoteJid na citação (o log [quote.inspect] mostra vazio): preenchido, aponta para outro chat.
func TestBuildTextMessage_QuoteLeavesRemoteJidEmpty(t *testing.T) {
	ci := buildTextMessage("ok", "Q1", "5511@lid", "oi", nil, nil, forwardMark{}).GetExtendedTextMessage().GetContextInfo()
	if ci.RemoteJID != nil || ci.GetParticipant() != "5511@lid" {
		t.Fatalf("remoteJid=%v participant=%q", ci.RemoteJID, ci.GetParticipant())
	}
	m := buildMediaMessage(MediaCommandPayload{MediaType: "image", QuotedMsgID: "Q1", QuotedJID: "5511@lid"}, whatsmeow.UploadResponse{}, mediameta.Meta{})
	if m.GetImageMessage().GetContextInfo().RemoteJID != nil {
		t.Fatal("citação de mídia também não preenche remoteJid")
	}
}

// Visualização única: a mídia (foto, vídeo e áudio) vai no envelope ViewOnceMessageV2 com a marca ViewOnce, SÓ quando pedida;
// ausente é o que o cliente oficial manda numa mídia comum. Documento e figurinha não têm esse modo e nunca são embrulhados,
// mesmo se o pedido chegar.
func TestBuildMediaMessage_ViewOnce(t *testing.T) {
	inner := func(m *waE2E.Message) *waE2E.Message { return m.GetViewOnceMessageV2().GetMessage() }
	flagOf := func(m *waE2E.Message) *bool {
		switch {
		case m.GetImageMessage() != nil:
			return m.GetImageMessage().ViewOnce
		case m.GetVideoMessage() != nil:
			return m.GetVideoMessage().ViewOnce
		case m.GetAudioMessage() != nil:
			return m.GetAudioMessage().ViewOnce
		}
		return nil
	}
	for _, mt := range []string{"image", "video", "audio"} {
		on := buildMediaMessage(MediaCommandPayload{MediaType: mt, ViewOnce: true}, whatsmeow.UploadResponse{}, mediameta.Meta{})
		if on.GetViewOnceMessageV2() == nil {
			t.Fatalf("%s com viewOnce: faltou o envelope ViewOnceMessageV2", mt)
		}
		if v := flagOf(inner(on)); v == nil || !*v {
			t.Errorf("%s com viewOnce: a mídia dentro do envelope precisa levar ViewOnce=true, veio %v", mt, v)
		}
		off := buildMediaMessage(MediaCommandPayload{MediaType: mt}, whatsmeow.UploadResponse{}, mediameta.Meta{})
		if off.GetViewOnceMessageV2() != nil {
			t.Errorf("%s comum: não pode vir embrulhada", mt)
		}
		if v := flagOf(off); v != nil {
			t.Errorf("%s comum: ViewOnce = %v, esperava ausente (nil)", mt, *v)
		}
	}
	// voz (mensagem de voz) também aceita, e continua sendo PTT
	voice := buildMediaMessage(MediaCommandPayload{MediaType: "voice", ViewOnce: true}, whatsmeow.UploadResponse{}, mediameta.Meta{})
	if a := inner(voice).GetAudioMessage(); a == nil || !a.GetPTT() || a.ViewOnce == nil || !*a.ViewOnce {
		t.Errorf("voz com viewOnce: %+v", a)
	}
	// documento e figurinha: o pedido é ignorado
	for _, mt := range []string{"document", "sticker"} {
		m := buildMediaMessage(MediaCommandPayload{MediaType: mt, ViewOnce: true}, whatsmeow.UploadResponse{}, mediameta.Meta{})
		if m.GetViewOnceMessageV2() != nil {
			t.Errorf("%s: não tem visualização única e não pode ser embrulhado", mt)
		}
		if m.GetDocumentMessage() == nil && m.GetStickerMessage() == nil {
			t.Fatalf("%s: tipo inesperado", mt)
		}
	}
}
