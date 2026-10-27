package services

import "testing"

func TestShouldAutoDownload(t *testing.T) {
	const max = 16 << 20
	cases := []struct {
		name     string
		typ      string
		length   int64
		max      int64
		hasProto bool
		want     bool
	}{
		{"sticker pequeno", "sticker", 45842, max, true, true},
		{"imagem", "image", 116941, max, true, true},
		{"audio", "audio", 20000, max, true, true},
		{"documento pequeno", "document", 1 << 20, max, true, true},
		{"documento acima do limite", "document", 20 << 20, max, true, false},
		{"video nunca", "video", 1000, max, true, false},
		{"texto", "chat", 10, max, true, false},
		{"sem proto", "image", 1000, max, false, false},
		{"tamanho desconhecido", "image", 0, max, true, false},
		{"recurso desligado", "image", 1000, 0, true, false},
	}
	for _, c := range cases {
		if got := shouldAutoDownload(c.typ, c.length, c.max, c.hasProto); got != c.want {
			t.Errorf("%s: %v, esperado %v", c.name, got, c.want)
		}
	}
}

func TestAutoDownloadMaxBytes_Env(t *testing.T) {
	t.Setenv("WATINK_MEDIA_AUTODOWNLOAD_MAX_BYTES", "0")
	if autoDownloadMaxBytes() != 0 {
		t.Fatal("0 deve desligar")
	}
	t.Setenv("WATINK_MEDIA_AUTODOWNLOAD_MAX_BYTES", "lixo")
	if autoDownloadMaxBytes() != defaultAutoDownloadMaxBytes {
		t.Fatal("valor inválido volta ao padrão")
	}
}
