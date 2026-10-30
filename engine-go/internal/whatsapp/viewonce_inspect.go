package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// Diagnóstico da visualização única (Epic 11 do roadmap: validar a viabilidade ANTES de desenhar API ou tela).
// Ligado por WATINK_VIEWONCE_INSPECT=1. Registra só METADADOS (tipos, tamanhos, se os campos existem); nunca o conteúdo da
// mídia, nem chaves, nem hashes. A pergunta que isto responde: o WhatsApp entrega o payload completo (url/directPath +
// mediaKey) a este aparelho vinculado em algum momento, ou só o aviso "unavailable"?

func viewOnceInspectEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("WATINK_VIEWONCE_INSPECT")))
	return v == "1" || v == "true"
}

// inspectViewOnceUnavailable descreve o aviso <unavailable type="view_once"/>.
func inspectViewOnceUnavailable(v *events.UndecryptableMessage) string {
	return fmt.Sprintf("UNAVAILABLE id=%s chat=%s sender=%s fromMe=%v unavailable=%v tipo=%q decryptFail=%q ts=%s",
		v.Info.ID, v.Info.Chat, v.Info.Sender.ToNonAD(), v.Info.IsFromMe, v.IsUnavailable, string(v.UnavailableType),
		string(v.DecryptFailMode), v.Info.Timestamp.UTC().Format("15:04:05"))
}

// inspectViewOnceMessage descreve uma mensagem que chegou DECIFRADA e é (ou veio de) visualização única: se o envelope
// existia, se há mídia dentro, e se a mídia está completa o bastante para baixar (url/directPath + mediaKey + hash).
// Devolve "" quando a mensagem não tem nada a ver com visualização única.
func inspectViewOnceMessage(v *events.Message) string {
	raw := v.RawMessage
	wrapped := raw.GetViewOnceMessage() != nil || raw.GetViewOnceMessageV2() != nil || raw.GetViewOnceMessageV2Extension() != nil
	if !v.IsViewOnce && !wrapped {
		return ""
	}
	inner := unwrap(v.Message)
	return fmt.Sprintf("DECIFRADA id=%s chat=%s sender=%s fromMe=%v isViewOnce=%v v2=%v v2ext=%v envelope=%v conteudo=%s",
		v.Info.ID, v.Info.Chat, v.Info.Sender.ToNonAD(), v.Info.IsFromMe, v.IsViewOnce, v.IsViewOnceV2, v.IsViewOnceV2Extension, wrapped,
		describeViewOnceMedia(inner))
}

// describeViewOnceMedia diz o tipo da mídia e se ela está completa para baixar, sem expor nenhum valor sensível.
func describeViewOnceMedia(m *waE2E.Message) string {
	has := func(b bool) string {
		if b {
			return "sim"
		}
		return "nao"
	}
	switch {
	case m.GetImageMessage() != nil:
		i := m.GetImageMessage()
		return fmt.Sprintf("imagem(mime=%q bytes=%d url=%s directPath=%s mediaKey=%s sha256=%s encSha256=%s miniatura=%dB viewOnce=%v)",
			i.GetMimetype(), i.GetFileLength(), has(i.GetURL() != ""), has(i.GetDirectPath() != ""), has(len(i.GetMediaKey()) > 0),
			has(len(i.GetFileSHA256()) > 0), has(len(i.GetFileEncSHA256()) > 0), len(i.GetJPEGThumbnail()), i.GetViewOnce())
	case m.GetVideoMessage() != nil:
		vd := m.GetVideoMessage()
		return fmt.Sprintf("video(mime=%q bytes=%d seg=%d url=%s directPath=%s mediaKey=%s sha256=%s encSha256=%s miniatura=%dB viewOnce=%v)",
			vd.GetMimetype(), vd.GetFileLength(), vd.GetSeconds(), has(vd.GetURL() != ""), has(vd.GetDirectPath() != ""), has(len(vd.GetMediaKey()) > 0),
			has(len(vd.GetFileSHA256()) > 0), has(len(vd.GetFileEncSHA256()) > 0), len(vd.GetJPEGThumbnail()), vd.GetViewOnce())
	case m.GetAudioMessage() != nil:
		a := m.GetAudioMessage()
		return fmt.Sprintf("audio(mime=%q bytes=%d seg=%d url=%s directPath=%s mediaKey=%s sha256=%s encSha256=%s viewOnce=%v)",
			a.GetMimetype(), a.GetFileLength(), a.GetSeconds(), has(a.GetURL() != ""), has(a.GetDirectPath() != ""), has(len(a.GetMediaKey()) > 0),
			has(len(a.GetFileSHA256()) > 0), has(len(a.GetFileEncSHA256()) > 0), a.GetViewOnce())
	case m == nil:
		return "vazio"
	}
	return "outro"
}

// inspectViewOnceContext descreve a CITAÇÃO que aponta para uma mensagem (a resposta do celular principal): se o
// ContextInfo traz a mídia citada com os campos de download. É o caminho que a dica do roadmap propõe investigar.
func inspectViewOnceContext(msg *waE2E.Message) string {
	msg = unwrap(msg)
	var ci *waE2E.ContextInfo
	switch {
	case msg.GetExtendedTextMessage() != nil:
		ci = msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		ci = msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		ci = msg.GetVideoMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		ci = msg.GetAudioMessage().GetContextInfo()
	}
	q := ci.GetQuotedMessage()
	if ci.GetStanzaID() == "" || q == nil {
		return ""
	}
	qv := q.GetViewOnceMessage() != nil || q.GetViewOnceMessageV2() != nil || q.GetViewOnceMessageV2Extension() != nil
	if !qv {
		// citação comum de uma mídia que o próprio contexto marca como visualização única
		inner := unwrap(q)
		if !inner.GetImageMessage().GetViewOnce() && !inner.GetVideoMessage().GetViewOnce() && !inner.GetAudioMessage().GetViewOnce() {
			return ""
		}
	}
	return fmt.Sprintf("CITACAO alvo=%s envelope=%v citado=%s", ci.GetStanzaID(), qv, describeViewOnceMedia(unwrap(q)))
}

func logViewOnceInspect(prefix, desc string) {
	if desc == "" {
		return
	}
	log.Printf("[viewonce.inspect] %s %s", prefix, strings.TrimSpace(desc))
}

// ---- Recuperação da visualização única citada pelo celular principal ----
//
// Quando o operador responde/cita a mensagem no aparelho PRINCIPAL, a citação traz os campos completos de download (provado
// com mídia de teste: o arquivo baixa íntegro, com o tamanho declarado e os hashes conferidos). O engine então baixa a mídia
// e a entrega ao business pelo evento message.media já existente, apontando para a mensagem de AVISO (view_once) original.
// O download é invisível para o remetente: não gera aviso de "visualizada" (conferido no experimento).

// viewOnceDownloader abstrai o cliente para o teste não depender de rede.
type viewOnceDownloader interface {
	Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error)
}

// quotedViewOnceMedia devolve a mídia citada quando ela é de visualização única, o id da mensagem de aviso e o tipo.
func quotedViewOnceMedia(msg *waE2E.Message) (whatsmeow.DownloadableMessage, string, string) {
	msg = unwrap(msg)
	var ci *waE2E.ContextInfo
	switch {
	case msg.GetExtendedTextMessage() != nil:
		ci = msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		ci = msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		ci = msg.GetVideoMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		ci = msg.GetAudioMessage().GetContextInfo()
	}
	q := unwrap(ci.GetQuotedMessage())
	if ci.GetStanzaID() == "" || q == nil {
		return nil, "", ""
	}
	switch {
	case q.GetImageMessage() != nil && q.GetImageMessage().GetViewOnce():
		return q.GetImageMessage(), ci.GetStanzaID(), "image"
	case q.GetVideoMessage() != nil && q.GetVideoMessage().GetViewOnce():
		return q.GetVideoMessage(), ci.GetStanzaID(), "video"
	case q.GetAudioMessage() != nil && q.GetAudioMessage().GetViewOnce():
		return q.GetAudioMessage(), ci.GetStanzaID(), "audio"
	}
	return nil, "", ""
}

// mimeOfViewOnce é o tipo declarado da mídia citada.
func mimeOfViewOnce(m whatsmeow.DownloadableMessage) string {
	switch x := m.(type) {
	case *waE2E.ImageMessage:
		return x.GetMimetype()
	case *waE2E.VideoMessage:
		return x.GetMimetype()
	case *waE2E.AudioMessage:
		return x.GetMimetype()
	}
	return ""
}

// declaredSize é o tamanho que o remetente declarou, para conferir com o que veio.
func declaredSize(m whatsmeow.DownloadableMessage) uint64 {
	switch x := m.(type) {
	case *waE2E.ImageMessage:
		return x.GetFileLength()
	case *waE2E.VideoMessage:
		return x.GetFileLength()
	case *waE2E.AudioMessage:
		return x.GetFileLength()
	}
	return 0
}

func classifyDownloadError(err error) string {
	switch {
	case errors.Is(err, whatsmeow.ErrNoURLPresent):
		return "sem-url-nem-directPath"
	case errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith403):
		return "http-403"
	case errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith404):
		return "http-404"
	case errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith410):
		return "http-410"
	case errors.Is(err, whatsmeow.ErrInvalidMediaSHA256), errors.Is(err, whatsmeow.ErrInvalidMediaEncSHA256):
		return "hash-invalido"
	}
	return "erro: " + err.Error()
}

// recoverViewOnce baixa a mídia citada e a publica sobre a mensagem de aviso. Só publica o que baixou ÍNTEGRO: o whatsmeow
// confere sha256 e encSha256, e o tamanho precisa bater com o declarado. Falha não publica nada (o aviso continua como estava).
// O arquivo nunca vai para o log.
func (s *WhatsAppService) recoverViewOnce(ctx context.Context, dl viewOnceDownloader, tenantID string, sessionID int, media whatsmeow.DownloadableMessage, target, kind string) {
	data, err := dl.Download(ctx, media)
	if err != nil {
		log.Printf("[Session-%d] visualização única %s: download falhou (%s); o aviso continua sem a mídia", sessionID, target, classifyDownloadError(err))
		return
	}
	if want := declaredSize(media); want > 0 && uint64(len(data)) != want {
		log.Printf("[Session-%d] visualização única %s: tamanho recebido %d difere do declarado %d; descartado", sessionID, target, len(data), want)
		return
	}
	log.Printf("[Session-%d] visualização única %s (%s) recuperada: %d bytes", sessionID, target, kind, len(data))
	s.publishEvent(tenantID, sessionID, "message.media", map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", sessionID),
		"messageId": target,
		"mediaData": base64.StdEncoding.EncodeToString(data),
		"mimetype":  mimeOfViewOnce(media),
		"mediaType": kind,
		"viewOnce":  true,
		"error":     "",
	})
}
