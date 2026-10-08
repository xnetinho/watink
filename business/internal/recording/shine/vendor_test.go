package mp3

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVendoredDiffIsOnlyBitrate prova que esta cópia difere da origem SÓ pela
// alteração documentada no NOTICE.md (NewEncoderBitrate). Precisa do clone do
// shine-mp3 no commit registrado, apontado por SHINE_ORIGIN_DIR; sem ele, o teste
// é pulado (não há como comparar sem a origem). Rode-o ao atualizar a cópia.
//
// Compara em Go puro (sem depender do `diff` do sistema): cada arquivo .go da
// origem deve existir aqui; todos, exceto layer3.go, idênticos byte a byte; e
// layer3.go, idêntico depois de desfeitas as duas mudanças conhecidas.
func TestVendoredDiffIsOnlyBitrate(t *testing.T) {
	origin := os.Getenv("SHINE_ORIGIN_DIR")
	if origin == "" {
		t.Skip("SHINE_ORIGIN_DIR não definido (clone do shine-mp3 em 517c45581c50c0cb44628987d38b975c49facd2f)")
	}
	originFiles, err := filepath.Glob(filepath.Join(origin, "pkg", "mp3", "*.go"))
	if err != nil || len(originFiles) == 0 {
		t.Fatalf("nenhum arquivo em %s/pkg/mp3: %v", origin, err)
	}
	for _, of := range originFiles {
		name := filepath.Base(of)
		want, _ := os.ReadFile(of)
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("falta %s na cópia: %v", name, err)
		}
		if name != "layer3.go" {
			if string(got) != string(want) {
				t.Errorf("%s difere da origem; só layer3.go pode diferir", name)
			}
			continue
		}
		undone := strings.Replace(string(got), `func NewEncoder(sampleRate, channels int) *Encoder {
	return NewEncoderBitrate(sampleRate, channels, 128)
}

// NewEncoderBitrate é a ÚNICA alteração do Watink em relação ao shine-mp3
// original: aceita o bitrate (kbps) como parâmetro em vez de fixar 128. O
// original só expõe o NewEncoder acima, que segue idêntico em comportamento.
func NewEncoderBitrate(sampleRate, channels, bitrate int) *Encoder {
`, "func NewEncoder(sampleRate, channels int) *Encoder {\n", 1)
		undone = strings.Replace(undone, "enc.Mpeg.Bitrate = int64(bitrate)", "enc.Mpeg.Bitrate = 128", 1)
		if undone != string(want) {
			t.Errorf("layer3.go difere da origem além do NewEncoderBitrate documentado")
		}
	}
	ours, _ := filepath.Glob("*.go")
	for _, f := range ours {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if _, err := os.Stat(filepath.Join(origin, "pkg", "mp3", f)); err != nil {
			t.Errorf("%s existe aqui mas não na origem", f)
		}
	}
}
