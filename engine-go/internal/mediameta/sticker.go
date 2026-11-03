package mediameta

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// StickerSide é o lado da figurinha: o WhatsApp pede exatamente 512x512.
	StickerSide = 512
	// MaxStaticSticker e MaxAnimatedSticker são os tamanhos máximos do arquivo (regras oficiais de figurinha do WhatsApp).
	MaxStaticSticker   = 100 << 10
	MaxAnimatedSticker = 500 << 10
	// MaxStickerSeconds é a duração máxima de uma figurinha animada.
	MaxStickerSeconds = 10
)

// Sticker é uma figurinha pronta para enviar: WebP 512x512, com a marca de animada.
type Sticker struct {
	Data     []byte
	Animated bool
}

// stickerStep é um degrau da escada de qualidade: tenta o melhor primeiro e desce até caber no limite.
type stickerStep struct {
	Quality int
	FPS     int // só animada
}

var (
	staticSteps   = []stickerStep{{80, 0}, {60, 0}, {40, 0}, {25, 0}, {10, 0}}
	animatedSteps = []stickerStep{{60, 15}, {45, 12}, {30, 10}, {20, 10}, {10, 8}, {5, 6}}
)

// stickerFilter leva a imagem para um quadrado 512x512 sem esticar nem cortar: reduz mantendo a proporção e centraliza
// sobre fundo transparente.
func stickerFilter(fps int) string {
	f := ""
	if fps > 0 {
		f = fmt.Sprintf("fps=%d,", fps)
	}
	return f + fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=0x00000000,format=yuva420p",
		StickerSide, StickerSide, StickerSide, StickerSide)
}

// ToSticker converte uma imagem, GIF, WebP ou vídeo curto em figurinha. animated diz se a ORIGEM tem movimento (mais de
// um quadro). Uma figurinha que já está certa (WebP 512x512 dentro do limite, estático ou animado) NÃO é recodificada:
// recodificar só perderia qualidade.
func ToSticker(ctx context.Context, data []byte, ext string) (Sticker, error) {
	if len(data) == 0 {
		return Sticker{}, errors.New("mediameta: figurinha vazia")
	}
	if info, ok := ParseWebP(data); ok && info.Width == StickerSide && info.Height == StickerSide {
		limit := MaxStaticSticker
		if info.Animated {
			limit = MaxAnimatedSticker
		}
		if len(data) <= limit {
			return Sticker{Data: data, Animated: info.Animated}, nil
		}
	}

	var res Sticker
	err := withTemp(data, ext, func(path string) error {
		if _, err := probe(ctx, path); err != nil {
			return err // não é mídia que o ffmpeg entenda
		}
		frames := countFrames(ctx, path)
		animated := frames > 1
		res.Animated = animated

		steps, limit, codec := staticSteps, MaxStaticSticker, "libwebp"
		args := func(q, fps int) []string {
			return []string{"-v", "error", "-y", "-i", path, "-an", "-vf", stickerFilter(fps), "-frames:v", "1", "-c:v", codec,
				"-q:v", fmt.Sprint(q), "-f", "webp", "-"}
		}
		if animated {
			steps, limit, codec = animatedSteps, MaxAnimatedSticker, "libwebp_anim"
			args = func(q, fps int) []string {
				return []string{"-v", "error", "-y", "-i", path, "-an", "-t", fmt.Sprint(MaxStickerSeconds), "-vf", stickerFilter(fps),
					"-c:v", codec, "-q:v", fmt.Sprint(q), "-loop", "0", "-f", "webp", "-"}
			}
		}
		var last []byte
		for _, s := range steps {
			out, err := run(ctx, convertTimeout, "ffmpeg", args(s.Quality, s.FPS)...)
			if err != nil {
				return err
			}
			last = out
			if len(out) > 0 && len(out) <= limit {
				res.Data = out
				return nil
			}
		}
		return fmt.Errorf("mediameta: figurinha continua com %d bytes no menor degrau (limite %d)", len(last), limit)
	})
	return res, err
}

// countFrames diz quantos quadros a origem tem (1 = estática). Em falha assume 1: na dúvida, estática.
func countFrames(ctx context.Context, path string) int {
	out, err := run(ctx, probeTimeout, "ffprobe", "-v", "error", "-count_frames", "-select_streams", "v:0",
		"-show_entries", "stream=nb_read_frames", "-of", "csv=p=0", path)
	if err != nil {
		return 1
	}
	var n int
	if _, err := fmt.Sscanf(string(out), "%d", &n); err != nil || n < 1 {
		return 1
	}
	return n
}

// WebPInfo é o que se lê do cabeçalho de um arquivo WebP, sem decodificar a imagem.
type WebPInfo struct {
	Width, Height int
	Animated      bool
}

// ParseWebP lê as dimensões e se é animado (chunk ANIM) direto do contêiner RIFF. Devolve ok=false para o que não é WebP.
func ParseWebP(b []byte) (WebPInfo, bool) {
	if len(b) < 20 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return WebPInfo{}, false
	}
	var info WebPInfo
	for i := 12; i+8 <= len(b); {
		tag, n := string(b[i:i+4]), int(binary.LittleEndian.Uint32(b[i+4:i+8]))
		body := b[i+8:]
		switch tag {
		case "VP8X":
			if len(body) >= 10 {
				info.Animated = body[0]&0x02 != 0
				info.Width = 1 + (int(body[4]) | int(body[5])<<8 | int(body[6])<<16)
				info.Height = 1 + (int(body[7]) | int(body[8])<<8 | int(body[9])<<16)
			}
		case "VP8 ":
			if info.Width == 0 && len(body) >= 10 {
				info.Width, info.Height = int(binary.LittleEndian.Uint16(body[6:8]))&0x3fff, int(binary.LittleEndian.Uint16(body[8:10]))&0x3fff
			}
		case "VP8L":
			if info.Width == 0 && len(body) >= 5 {
				v := binary.LittleEndian.Uint32(body[1:5])
				info.Width, info.Height = int(v&0x3fff)+1, int((v>>14)&0x3fff)+1
			}
		}
		if n < 0 || i+8+n > len(b) {
			break
		}
		i += 8 + n + (n & 1)
	}
	return info, info.Width > 0
}
