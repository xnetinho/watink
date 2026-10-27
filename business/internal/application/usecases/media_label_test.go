package usecases

import "testing"

func TestMediaTypeLabel(t *testing.T) {
	for typ, want := range map[string]string{
		"view_once": "👁 Visualização única", "location": "📍 Localização", "vcard": "👤 Contato",
		"unsupported": "⚠️ Mensagem não suportada", "chat": "", "image": "📷 Foto", "poll": "📊 Enquete", "event": "📅 Evento", "pix": "💠 PIX",
	} {
		if got := MediaTypeLabel(typ); got != want {
			t.Errorf("%q: %q, esperado %q", typ, got, want)
		}
	}
}
