package whatsapp

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
)

// extOf escolhe a extensão que o ffmpeg usa para reconhecer o contêiner: pelo nome do arquivo e, na falta, pelo mimetype.
func extOf(payload MediaCommandPayload) string {
	if e := strings.TrimPrefix(filepath.Ext(payload.FileName), "."); e != "" {
		return e
	}
	if e := strings.TrimPrefix(filepath.Ext(payload.MediaURL), "."); e != "" {
		return e
	}
	if i := strings.IndexByte(payload.MimeType, '/'); i >= 0 {
		return strings.SplitN(payload.MimeType[i+1:], ";", 2)[0]
	}
	return ""
}

// enrichMedia devolve os bytes a enviar (convertidos para Opus quando é gravação de voz), o payload ajustado e os
// metadados. Sem ffmpeg ou com mídia que ele não entende, devolve a mídia original e metadados vazios.
func enrichMedia(ctx context.Context, data []byte, payload MediaCommandPayload) ([]byte, MediaCommandPayload, mediameta.Meta) {
	ext := extOf(payload)
	// PDF usa o poppler, não o ffmpeg: o ramo vem antes da checagem do ffmpeg.
	if strings.EqualFold(payload.MediaType, "document") && isPDF(payload, ext) {
		return data, payload, mediameta.PDF(ctx, data)
	}
	if !mediameta.Available() {
		return data, payload, mediameta.Meta{}
	}
	switch strings.ToLower(payload.MediaType) {
	case "sticker":
		// Figurinha precisa ser WebP 512x512 dentro do limite de tamanho. Imagem, GIF e vídeo curto são convertidos aqui;
		// se a conversão falhar a mídia NÃO segue como figurinha torta: o erro sobe e o envio falha com aviso.
		st, err := mediameta.ToSticker(ctx, data, ext)
		if err != nil {
			log.Printf("[media] figurinha %s: conversão falhou: %v", payload.MessageID, err)
			payload.ConvertError = "figurinha: " + err.Error()
			return data, payload, mediameta.Meta{}
		}
		payload.MimeType = "image/webp"
		meta := mediameta.Image(ctx, st.Data, "webp")
		// O ffmpeg não lê WebP animado (largura/altura/miniatura vêm vazias), mas o cliente oficial manda as medidas: lê do
		// cabeçalho do próprio WebP, que sempre as tem.
		if info, ok := mediameta.ParseWebP(st.Data); ok {
			meta.Width, meta.Height = info.Width, info.Height
		}
		meta.Animated = st.Animated
		return st.Data, payload, meta
	case "image":
		// O WhatsApp não tem GIF como imagem: um ImageMessage image/gif é aceito pelo servidor e descartado pelo celular. O GIF
		// vira vídeo MP4 com gifPlayback, como o cliente oficial envia. Se a conversão falhar o envio falha com aviso.
		if mediameta.IsGIF(data) {
			gif, err := mediameta.GIFToVideo(ctx, data)
			if err != nil {
				log.Printf("[media] gif %s: conversão falhou: %v", payload.MessageID, err)
				payload.ConvertError = "gif: " + err.Error()
				return data, payload, mediameta.Meta{}
			}
			payload.MediaType, payload.MimeType = "video", "video/mp4"
			meta := mediameta.Video(ctx, gif.Data, "mp4")
			meta.GIF, meta.GIFTrimmed = true, gif.Trimmed
			return gif.Data, payload, meta
		}
		return data, payload, mediameta.Image(ctx, data, ext)
	case "video":
		return data, payload, mediameta.Video(ctx, data, ext)
	case "voice":
		// Mensagem de voz do WhatsApp é OGG/Opus. Se a conversão falhar, vai como arquivo de áudio (não some).
		ogg, err := mediameta.ToOpusVoice(ctx, data, ext)
		if err != nil {
			log.Printf("[media] conversão para Opus falhou (%s); enviando como arquivo de áudio: %v", payload.MessageID, err)
			payload.MediaType = "audio"
			return data, payload, mediameta.Audio(ctx, data, ext)
		}
		payload.MimeType = "audio/ogg; codecs=opus"
		return ogg, payload, mediameta.Audio(ctx, ogg, "ogg")
	case "audio":
		payload.MimeType = normalizeAudioMime(payload.MimeType)
		m := mediameta.Audio(ctx, data, ext)
		// Arquivo de áudio NUNCA leva forma de onda, em nenhum formato: medido com o cliente oficial em 29/out/2026,
		// .opus (audio/ogg; codecs=opus), .aac (audio/aac) e .m4a (audio/mp4) chegaram todos com ptt=false waveform=0B.
		// A onda é só da gravação (voice). Mandá-la num arquivo é o que faz o celular desenhá-lo como mensagem de voz.
		m.Waveform = nil
		return data, payload, m
	}
	return data, payload, mediameta.Meta{}
}

// isPDF reconhece PDF pelo mimetype ou pela extensão.
func isPDF(payload MediaCommandPayload, ext string) bool {
	return strings.EqualFold(payload.MimeType, "application/pdf") || strings.EqualFold(ext, "pdf")
}

// normalizeAudioMime corrige o mimetype de OGG: "audio/ogg" puro NÃO é entregue pelo WhatsApp (0 recibos nos testes de
// 28/out/2026); o cliente oficial manda "audio/ogg; codecs=opus". Qualquer outro mimetype passa como veio.
func normalizeAudioMime(mime string) string {
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(mime, ";", 2)[0]))
	if (base == "audio/ogg" || base == "application/ogg" || base == "audio/opus") && !strings.Contains(strings.ToLower(mime), "codecs=") {
		return "audio/ogg; codecs=opus"
	}
	return mime
}
