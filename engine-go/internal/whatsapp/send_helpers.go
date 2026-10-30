package whatsapp

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// forwardMark é a marca de "encaminhada" no ContextInfo; score é quantas vezes a mensagem já foi encaminhada.
type forwardMark struct {
	forwarded bool
	score     uint32
}

func forwardInfo(forwarded bool, score int) forwardMark {
	if !forwarded {
		return forwardMark{}
	}
	if score < 1 {
		score = 1
	}
	return forwardMark{forwarded: true, score: uint32(score)}
}

func (f forwardMark) apply(ctx *waProto.ContextInfo) {
	if f.forwarded {
		ctx.IsForwarded = proto.Bool(true)
		ctx.ForwardingScore = proto.Uint32(f.score)
	}
}

func buildTextMessage(body, quotedMsgID, quotedJID, quotedBody string, quotedMedia *QuotedMedia, mentions []string, fwd forwardMark) *waProto.Message {
	if quotedMsgID == "" && len(mentions) == 0 && !fwd.forwarded {
		return &waProto.Message{Conversation: proto.String(body)}
	}

	ctx := &waProto.ContextInfo{}
	fwd.apply(ctx)
	if quotedMsgID != "" {
		ctx.StanzaID = proto.String(quotedMsgID)
		if quotedJID != "" {
			ctx.Participant = proto.String(quotedJID)
		}
		ctx.QuotedMessage = quotedContent(quotedBody, quotedMedia)
	}
	if len(mentions) > 0 {
		ctx.MentionedJID = mentions
	}

	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(body),
			ContextInfo: ctx,
		},
	}
}

// quotedContent é o conteúdo da mensagem citada que o celular usa para desenhar a caixa de citação. Só o
// texto: citar mídia mostra a legenda (ou o rótulo que o business mandou). nil sem corpo — cita só pelo id.
func quotedContent(body string, qm *QuotedMedia) *waProto.Message {
	if m := quotedMediaContent(qm); m != nil {
		return m
	}
	if body == "" {
		return nil
	}
	return &waProto.Message{Conversation: proto.String(body)}
}

// quotedMediaContent monta a mensagem de mídia que vai DENTRO da citação, como o cliente oficial: o tipo certo, com
// duração/voz (áudio) ou miniatura e legenda (imagem, vídeo, documento). nil quando não há mídia a citar.
func quotedMediaContent(qm *QuotedMedia) *waProto.Message {
	if qm == nil {
		return nil
	}
	var thumb []byte
	if qm.Thumbnail != "" {
		thumb, _ = base64.StdEncoding.DecodeString(qm.Thumbnail)
	}
	switch qm.Type {
	case "audio":
		a := &waProto.AudioMessage{Mimetype: proto.String(normalizeAudioMime(qm.MimeType)), PTT: proto.Bool(qm.Voice)}
		if qm.Seconds > 0 {
			a.Seconds = proto.Uint32(uint32(qm.Seconds))
		}
		return &waProto.Message{AudioMessage: a}
	case "image":
		return &waProto.Message{ImageMessage: &waProto.ImageMessage{Mimetype: proto.String(qm.MimeType), Caption: proto.String(qm.Caption), JPEGThumbnail: thumb}}
	case "video":
		return &waProto.Message{VideoMessage: &waProto.VideoMessage{Mimetype: proto.String(qm.MimeType), Caption: proto.String(qm.Caption), JPEGThumbnail: thumb}}
	case "document":
		return &waProto.Message{DocumentMessage: &waProto.DocumentMessage{Mimetype: proto.String(qm.MimeType), FileName: proto.String(qm.FileName), Title: proto.String(qm.FileName), Caption: proto.String(qm.Caption), JPEGThumbnail: thumb}}
	}
	return nil
}

func resolveMediaBytes(payload MediaCommandPayload) ([]byte, error) {
	if payload.MediaData != "" {
		return base64.StdEncoding.DecodeString(payload.MediaData)
	}
	if payload.MediaURL == "" {
		return nil, fmt.Errorf("mediaUrl or mediaData is required")
	}
	if strings.HasPrefix(payload.MediaURL, "http://") || strings.HasPrefix(payload.MediaURL, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(payload.MediaURL)
		if err != nil {
			return nil, fmt.Errorf("download media: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("download media: HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	paths := []string{payload.MediaURL}
	if !filepath.IsAbs(payload.MediaURL) {
		paths = append(paths, filepath.Join("public", payload.MediaURL))
		paths = append(paths, filepath.Join("..", "business", "public", payload.MediaURL))
	}
	var lastErr error
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// resolveImageBytes fetches image bytes from a URL or base64 string, reusing the
// same resolution logic as media messages (http, local public dir, or base64).
func resolveImageBytes(url, b64 string) ([]byte, error) {
	return resolveMediaBytes(MediaCommandPayload{MediaURL: url, MediaData: b64})
}

func normalizeMediaType(mediaType string) whatsmeow.MediaType {
	switch strings.ToLower(mediaType) {
	case "image":
		return whatsmeow.MediaImage
	case "video":
		return whatsmeow.MediaVideo
	case "audio", "voice":
		return whatsmeow.MediaAudio
	case "sticker":
		return whatsmeow.MediaImage
	default:
		return whatsmeow.MediaDocument
	}
}

// viewOnceFlag liga a visualização única só quando pedida: ausente é o que o cliente oficial manda numa mídia comum.
func viewOnceFlag(p MediaCommandPayload) *bool {
	if !p.ViewOnce {
		return nil
	}
	return proto.Bool(true)
}

// optU32 devolve nil para 0: campo ausente no protobuf, como o cliente oficial faz quando não conhece o valor.
func optU32(v int) *uint32 {
	if v <= 0 {
		return nil
	}
	return proto.Uint32(uint32(v))
}

func buildMediaPayload(payload MediaCommandPayload, uploaded whatsmeow.UploadResponse, meta mediameta.Meta) *waE2E.Message {
	mimeType := payload.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var ctx *waProto.ContextInfo
	fwd := forwardInfo(payload.Forwarded, payload.ForwardScore)
	if payload.QuotedMsgID != "" || len(payload.Mentions) > 0 || fwd.forwarded {
		ctx = &waProto.ContextInfo{}
		fwd.apply(ctx)
		if payload.QuotedMsgID != "" {
			ctx.StanzaID = proto.String(payload.QuotedMsgID)
			if payload.QuotedJID != "" {
				ctx.Participant = proto.String(payload.QuotedJID)
			}
			ctx.QuotedMessage = quotedContent(payload.QuotedBody, payload.QuotedMedia)
		}
		if len(payload.Mentions) > 0 {
			ctx.MentionedJID = payload.Mentions
		}
	}

	switch strings.ToLower(payload.MediaType) {
	case "image":
		return &waProto.Message{ImageMessage: &waProto.ImageMessage{
			ViewOnce: viewOnceFlag(payload), Caption: proto.String(payload.Body), Mimetype: proto.String(mimeType),
			Height: optU32(meta.Height), Width: optU32(meta.Width), JPEGThumbnail: meta.Thumbnail,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			URL:               proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
			MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			ContextInfo: ctx,
		}}
	case "video":
		return &waProto.Message{VideoMessage: &waProto.VideoMessage{
			ViewOnce: viewOnceFlag(payload), Caption: proto.String(payload.Body), Mimetype: proto.String(mimeType),
			Seconds: optU32(meta.Seconds), Height: optU32(meta.Height), Width: optU32(meta.Width), JPEGThumbnail: meta.Thumbnail,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			URL:               proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
			MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			ContextInfo: ctx,
		}}
	case "audio", "voice":
		voice := strings.EqualFold(payload.MediaType, "voice")
		am := &waProto.AudioMessage{
			ViewOnce: viewOnceFlag(payload), Mimetype: proto.String(mimeType),
			Seconds: optU32(meta.Seconds), MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			// Onda só quando existe (Opus). O cliente oficial NÃO manda onda em MP3 de arquivo (waveform=0B no log).
			Waveform: meta.Waveform,
			// PTT sempre EXPLÍCITO: ausente faz o celular desenhar o áudio como mensagem de voz; o cliente oficial manda
			// ptt=false no arquivo de áudio.
			PTT: proto.Bool(voice),
			URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
			MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			ContextInfo: ctx,
		}
		return &waProto.Message{AudioMessage: am}
	case "sticker":
		return &waProto.Message{StickerMessage: &waProto.StickerMessage{
			Mimetype: proto.String(mimeType),
			URL:      proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
			MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			ContextInfo: ctx,
		}}
	default:
		fileName := payload.FileName
		if fileName == "" {
			fileName = filepath.Base(payload.MediaURL)
		}
		return &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
			Caption: proto.String(payload.Body), Title: proto.String(fileName),
			FileName: proto.String(fileName), Mimetype: proto.String(mimeType),
			PageCount: optU32(meta.Pages), JPEGThumbnail: meta.Thumbnail,
			ThumbnailWidth: optU32(meta.ThumbW), ThumbnailHeight: optU32(meta.ThumbH),
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			URL:               proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
			MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			ContextInfo: ctx,
		}}
	}
}
