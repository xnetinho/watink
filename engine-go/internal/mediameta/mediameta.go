// Package mediameta gera o que um cliente oficial do WhatsApp envia junto com a mídia e uma biblioteca costuma omitir:
// miniatura JPEG, largura/altura, duração, forma de onda do áudio e a conversão do áudio para Opus (mensagem de voz).
// Usa ffmpeg/ffprobe (presentes na imagem do engine). É sempre de MELHOR ESFORÇO: se o ffmpeg faltar, travar ou a mídia
// for inválida, o envio segue sem os metadados em vez de falhar.
package mediameta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// ThumbMaxSide é o maior lado da miniatura JPEG (o WhatsApp usa ~100 px para imagem/vídeo).
	ThumbMaxSide = 100
	// WaveformSamples é a quantidade de barras da forma de onda (o cliente oficial manda 64).
	WaveformSamples = 64

	probeTimeout   = 20 * time.Second
	convertTimeout = 90 * time.Second
	maxInputBytes  = 120 << 20
)

// Meta reúne os metadados de uma mídia. Zero/nil significa "não obtido".
type Meta struct {
	Width, Height  int
	Seconds        int
	Pages          int    // páginas (PDF)
	ThumbW, ThumbH int    // dimensões da miniatura JPEG (documento)
	Thumbnail      []byte // JPEG
	Waveform       []byte // 64 amostras 0..100
}

// AvailablePDF diz se o poppler (pdfinfo/pdftoppm) existe. Sem ele, PDF segue sem miniatura e sem nº de páginas.
func AvailablePDF() bool {
	_, e1 := exec.LookPath("pdfinfo")
	_, e2 := exec.LookPath("pdftoppm")
	return e1 == nil && e2 == nil
}

// Available diz se o ffmpeg e o ffprobe existem. Sem eles, tudo aqui devolve vazio.
func Available() bool {
	_, e1 := exec.LookPath("ffmpeg")
	_, e2 := exec.LookPath("ffprobe")
	return e1 == nil && e2 == nil
}

// withTemp grava os bytes em um arquivo temporário (o ffmpeg lê de arquivo, não de pipe, para formatos como MP4 cujo
// índice fica no fim do arquivo) e o remove ao terminar.
func withTemp(data []byte, ext string, fn func(path string) error) error {
	if len(data) == 0 {
		return errors.New("mediameta: mídia vazia")
	}
	if len(data) > maxInputBytes {
		return errors.New("mediameta: mídia grande demais para processar")
	}
	f, err := os.CreateTemp("", "mediameta-*"+safeExt(ext))
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return fn(f.Name())
}

// safeExt devolve só uma extensão alfanumérica curta: o nome do arquivo nunca carrega entrada do usuário.
func safeExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "" || len(ext) > 5 {
		return ".bin"
	}
	for _, r := range ext {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ".bin"
		}
	}
	return "." + ext
}

func run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(tail(errb.String(), 300)))
	}
	return out.Bytes(), nil
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

type probeOut struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
		Tags      struct {
			Rotate string `json:"rotate"`
		} `json:"tags"`
		SideDataList []struct {
			Rotation float64 `json:"rotation"`
		} `json:"side_data_list"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func probe(ctx context.Context, path string) (probeOut, error) {
	var p probeOut
	out, err := run(ctx, probeTimeout, "ffprobe", "-v", "error", "-print_format", "json", "-show_format", "-show_streams", path)
	if err != nil {
		return p, err
	}
	return p, json.Unmarshal(out, &p)
}

func seconds(p probeOut) int {
	for _, d := range []string{p.Format.Duration, firstDuration(p)} {
		if f, err := strconv.ParseFloat(d, 64); err == nil && f > 0 {
			// TRUNCA, como o cliente oficial (opus 21,88 s → 21; aac 4,28 s → 4; m4a 26,46 s → 26, medido em 29/out/2026),
			// mas nunca devolve 0 para uma mídia que existe: 0,4 s vira 1.
			s := int(f)
			if s < 1 {
				s = 1
			}
			return s
		}
	}
	return 0
}

func firstDuration(p probeOut) string {
	for _, s := range p.Streams {
		if s.Duration != "" {
			return s.Duration
		}
	}
	return ""
}

// size devolve largura/altura do primeiro fluxo de vídeo já com a rotação aplicada (vídeo de celular gravado em pé
// vem como paisagem com rotação de 90°; sem corrigir, a prévia sai deitada).
func size(p probeOut) (w, h int) {
	for _, s := range p.Streams {
		if s.CodecType != "video" || s.Width == 0 || s.Height == 0 {
			continue
		}
		w, h = s.Width, s.Height
		rot := 0.0
		if r, err := strconv.ParseFloat(s.Tags.Rotate, 64); err == nil {
			rot = r
		}
		for _, sd := range s.SideDataList {
			if sd.Rotation != 0 {
				rot = sd.Rotation
			}
		}
		if r := int(rot) % 180; r == 90 || r == -90 {
			w, h = h, w
		}
		return w, h
	}
	return 0, 0
}

// thumbArgs gera um JPEG com o maior lado em ThumbMaxSide, mantendo a proporção e sem ampliar.
func thumbArgs(in string, seek string) []string {
	scale := fmt.Sprintf("scale='if(gt(iw,ih),min(%d,iw),-2)':'if(gt(iw,ih),-2,min(%d,ih))'", ThumbMaxSide, ThumbMaxSide)
	args := []string{"-v", "error", "-y"}
	if seek != "" {
		args = append(args, "-ss", seek)
	}
	return append(args, "-i", in, "-frames:v", "1", "-vf", scale, "-q:v", "6", "-f", "image2pipe", "-vcodec", "mjpeg", "-")
}

// Image devolve dimensões e miniatura de uma imagem (JPEG, PNG, WebP...).
func Image(ctx context.Context, data []byte, ext string) Meta {
	var m Meta
	_ = withTemp(data, ext, func(path string) error {
		if p, err := probe(ctx, path); err == nil {
			m.Width, m.Height = size(p)
		}
		if th, err := run(ctx, probeTimeout, "ffmpeg", thumbArgs(path, "")...); err == nil && len(th) > 0 {
			m.Thumbnail = th
		}
		return nil
	})
	return m
}

// Video devolve dimensões, duração e miniatura de um quadro do início do vídeo.
func Video(ctx context.Context, data []byte, ext string) Meta {
	var m Meta
	_ = withTemp(data, ext, func(path string) error {
		p, err := probe(ctx, path)
		if err != nil {
			return err
		}
		m.Width, m.Height = size(p)
		m.Seconds = seconds(p)
		// Um quadro logo após o início (o primeiro costuma ser preto); em vídeos curtos, o próprio início.
		seek := "0.5"
		if m.Seconds <= 1 {
			seek = "0"
		}
		if th, err := run(ctx, probeTimeout, "ffmpeg", thumbArgs(path, seek)...); err == nil && len(th) > 0 {
			m.Thumbnail = th
		}
		return nil
	})
	return m
}

// Audio devolve a duração e a forma de onda (WaveformSamples barras, 0..100).
func Audio(ctx context.Context, data []byte, ext string) Meta {
	var m Meta
	_ = withTemp(data, ext, func(path string) error {
		if p, err := probe(ctx, path); err == nil {
			m.Seconds = seconds(p)
		}
		m.Waveform = waveform(ctx, path)
		return nil
	})
	return m
}

// waveform decodifica o áudio para PCM 8 kHz mono de 16 bits e reduz a WaveformSamples barras pelo pico de cada janela.
func waveform(ctx context.Context, path string) []byte {
	pcm, err := run(ctx, convertTimeout, "ffmpeg", "-v", "error", "-i", path, "-ac", "1", "-ar", "8000", "-f", "s16le", "-")
	if err != nil || len(pcm) < 2 {
		return nil
	}
	return waveformFromPCM(pcm)
}

func waveformFromPCM(pcm []byte) []byte {
	n := len(pcm) / 2
	if n == 0 {
		return nil
	}
	// Cada barra cobre uma janela de n/WaveformSamples amostras; a última leva o resto.
	win := n / WaveformSamples
	if win < 1 {
		win = 1
	}
	peaks := make([]float64, WaveformSamples)
	var max float64
	for i := range peaks {
		start := i * win
		if start >= n {
			break
		}
		end := start + win
		if i == WaveformSamples-1 {
			end = n
		}
		var peak float64
		for j := start; j < end; j++ {
			v := float64(int16(uint16(pcm[2*j]) | uint16(pcm[2*j+1])<<8))
			if v < 0 {
				v = -v
			}
			if v > peak {
				peak = v
			}
		}
		peaks[i] = peak
		if peak > max {
			max = peak
		}
	}
	out := make([]byte, WaveformSamples)
	if max == 0 {
		return out
	}
	for i, p := range peaks {
		out[i] = byte(p / max * 100)
	}
	return out
}

// ToOpusVoice converte qualquer áudio para OGG/Opus mono 48 kHz, o formato da mensagem de voz do WhatsApp (PTT). Devolve
// os bytes convertidos; em falha devolve erro e o chamador mantém o áudio original.
func ToOpusVoice(ctx context.Context, data []byte, ext string) ([]byte, error) {
	var out []byte
	err := withTemp(data, ext, func(path string) error {
		b, err := run(ctx, convertTimeout, "ffmpeg", "-v", "error", "-y", "-i", path, "-vn", "-ac", "1", "-ar", "48000",
			"-c:a", "libopus", "-b:a", "32k", "-application", "voip", "-f", "ogg", "-")
		if err != nil {
			return err
		}
		out = b
		return nil
	})
	if err == nil && len(out) == 0 {
		err = errors.New("mediameta: conversão para Opus não produziu saída")
	}
	return out, err
}

func readFile(p string) ([]byte, error) { return os.ReadFile(p) }

func writeFile(p string, b []byte) error { return os.WriteFile(p, b, 0o600) }

// DocThumbMaxSide é o maior lado da miniatura de documento (o cliente oficial manda 51x72 para uma página A4).
const DocThumbMaxSide = 72

var pagesRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

// PDF devolve o número de páginas e a miniatura (JPEG) da primeira página. Usa o poppler; PDF protegido por senha ou
// corrompido devolve o que conseguir (às vezes só um dos dois) em vez de falhar.
func PDF(ctx context.Context, data []byte) Meta {
	var m Meta
	if !AvailablePDF() {
		return m
	}
	_ = withTemp(data, "pdf", func(path string) error {
		if out, err := run(ctx, probeTimeout, "pdfinfo", path); err == nil {
			if g := pagesRe.FindSubmatch(out); g != nil {
				m.Pages, _ = strconv.Atoi(string(g[1]))
			}
		}
		// Saída em ARQUIVO, não em stdout: o poppler mistura avisos ("Couldn't find a font") com os bytes da imagem no
		// stdout, e um JPEG com texto no começo seria enviado corrompido. -scale-to limita o MAIOR lado: página em pé fica
		// com 72 de altura, paisagem com 72 de largura.
		dir, derr := os.MkdirTemp("", "mediameta-pdf-")
		if derr != nil {
			return derr
		}
		defer func() { _ = os.RemoveAll(dir) }()
		base := filepath.Join(dir, "p")
		if _, err := run(ctx, probeTimeout, "pdftoppm", "-jpeg", "-jpegopt", "quality=70", "-f", "1", "-l", "1",
			"-scale-to", strconv.Itoa(DocThumbMaxSide), "-singlefile", path, base); err == nil {
			if jpg, rerr := os.ReadFile(base + ".jpg"); rerr == nil && isJPEG(jpg) {
				m.Thumbnail = jpg
				if cfg, e := jpeg.DecodeConfig(bytes.NewReader(jpg)); e == nil {
					m.ThumbW, m.ThumbH = cfg.Width, cfg.Height
				}
			}
		}
		return nil
	})
	return m
}

// isJPEG confere a assinatura (FF D8 FF): nunca se envia como miniatura algo que não seja uma imagem.
func isJPEG(b []byte) bool { return len(b) > 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff }
