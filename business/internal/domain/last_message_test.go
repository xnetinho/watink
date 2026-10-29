package domain

import "testing"

func TestLastMessagePreview(t *testing.T) {
	cases := []struct {
		name, typ, body, mime, wantText, wantKind string
	}{
		{"texto comum", "chat", "oi tudo bem", "", "oi tudo bem", ""},
		{"imagem com legenda: a legenda vence", "image", "olha isso", "image/jpeg", "olha isso", ""},
		{"imagem sem legenda: marcador", "image", "", "image/jpeg", "", "image"},
		{"vídeo sem legenda", "video", "", "", "", "video"},
		{"áudio", "audio", "", "", "", "audio"},
		{"documento sem legenda", "document", "", "", "", "document"},
		{"figurinha", "sticker", "", "", "", "sticker"},
		{"localização: corpo são coordenadas, nunca vira texto", "location", "|https://maps/x|Praça", "", "", "location"},
		{"contato: corpo é vCard", "vcard", "BEGIN:VCARD...", "", "", "vcard"},
		{"enquete: corpo é texto formatado", "poll", "📊 Q?\n• A", "", "", "poll"},
		{"evento", "event", "📅 Reunião", "", "", "event"},
		{"pix", "pix", "💠 PIX: Meu PIX", "", "", "pix"},
		{"não suportada: corpo é nome de campo interno", "unsupported", "interactiveMessage", "", "", "unsupported"},
		{"visualização única", "view_once", "aviso", "", "", "view_once"},
		{"sem tipo, mas com mimetype de imagem", "", "", "image/png", "", "image"},
		{"sem tipo, mimetype qualquer vira documento", "", "", "application/pdf", "", "document"},
		{"nada", "", "", "", "", ""},
		{"só espaços não é texto", "chat", "   ", "", "", ""},
	}
	for _, c := range cases {
		text, kind := LastMessagePreview(c.typ, c.body, c.mime)
		if text != c.wantText || kind != c.wantKind {
			t.Errorf("%s: (%q,%q), esperado (%q,%q)", c.name, text, kind, c.wantText, c.wantKind)
		}
	}
}
