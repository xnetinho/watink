package mediameta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
)

const (
	// GIFMaxSide é o maior lado do vídeo gerado a partir de um GIF: o MP4 fica pequeno e o WhatsApp o toca como GIF.
	GIFMaxSide = 512
	// GIFMaxSeconds é a duração máxima de um GIF enviado: acima disso o vídeo é cortado.
	GIFMaxSeconds = 15
)

// GIFVideo é um GIF convertido no que o WhatsApp realmente envia como GIF: um vídeo MP4 curto, sem som, tocado em loop.
type GIFVideo struct {
	Data    []byte
	Trimmed bool // o GIF era mais longo que GIFMaxSeconds e foi cortado
}

// IsGIF reconhece o GIF pelos bytes (cabeçalho GIF87a/GIF89a), sem confiar no tipo informado pelo navegador.
func IsGIF(b []byte) bool {
	return bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a"))
}

// GIFToVideo converte um GIF em MP4 (H.264 baseline, yuv420p, dimensões pares, sem áudio, índice no início): o formato que
// qualquer celular toca. O WhatsApp NÃO tem GIF como imagem: um ImageMessage image/gif é aceito pelo servidor mas o aparelho
// do destinatário o descarta, sem aviso. O GIF do cliente oficial é um VideoMessage MP4 com gifPlayback ligado.
//
// O MP4 não tem canal de transparência: o GIF transparente fica sobre fundo preto. É a mesma limitação do cliente oficial.
func GIFToVideo(ctx context.Context, data []byte) (GIFVideo, error) {
	if !IsGIF(data) {
		return GIFVideo{}, errors.New("mediameta: não é um GIF")
	}
	var res GIFVideo
	err := withTemp(data, "gif", func(path string) error {
		p, err := probe(ctx, path)
		if err != nil {
			return err
		}
		res.Trimmed = gifDuration(ctx, path, p) > GIFMaxSeconds
		// Reduz o lado maior a GIFMaxSide sem ampliar, depois força dimensões pares (o libx264 recusa ímpares).
		scale := fmt.Sprintf("scale='if(gt(iw,ih),min(%d,iw),-2)':'if(gt(iw,ih),-2,min(%d,ih))',scale=trunc(iw/2)*2:trunc(ih/2)*2", GIFMaxSide, GIFMaxSide)
		// +faststart reescreve o índice no início e por isso precisa de um arquivo, não de um pipe.
		outPath := path + ".mp4"
		defer func() { _ = os.Remove(outPath) }()
		if _, err := run(ctx, convertTimeout, "ffmpeg", "-v", "error", "-y", "-i", path, "-t", fmt.Sprint(GIFMaxSeconds), "-an",
			"-vf", scale, "-pix_fmt", "yuv420p", "-c:v", "libx264", "-profile:v", "baseline", "-level", "3.0", "-crf", "23",
			"-movflags", "+faststart", outPath); err != nil {
			return err
		}
		out, err := readFile(outPath)
		if err != nil {
			return err
		}
		res.Data = out
		return nil
	})
	if err == nil && len(res.Data) == 0 {
		err = errors.New("mediameta: conversão do GIF não produziu saída")
	}
	return res, err
}

// gifDuration é a duração do GIF em segundos; um GIF sem duração informada conta como curto.
func gifDuration(ctx context.Context, path string, p probeOut) float64 {
	for _, d := range []string{p.Format.Duration, firstDuration(p)} {
		var f float64
		if _, err := fmt.Sscanf(d, "%f", &f); err == nil && f > 0 {
			return f
		}
	}
	return 0
}
