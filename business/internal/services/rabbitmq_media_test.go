package services

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
)

func sendMediaCmd(mediaURL string) map[string]interface{} {
	return map[string]interface{}{
		"type":    "message.send.media",
		"payload": map[string]interface{}{"sessionId": 1, "mediaUrl": mediaURL, "mediaType": "image"},
	}
}

func payloadOf(cmd map[string]interface{}) map[string]interface{} {
	return cmd["payload"].(map[string]interface{})
}

// O engine roda em outro container e não enxerga o disco do business: mandar só "/public/media/x.png"
// fazia o envio falhar com "no such file" e a imagem ficava no relógio. O comando precisa levar os bytes.
func TestInlineLocalMedia(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	url, err := mediastore.SaveMediaReader(strings.NewReader("PNGDATA"), "image/png")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("arquivo local vira mediaData em base64 e mantém o mediaUrl", func(t *testing.T) {
		cmd := sendMediaCmd(url)
		if err := inlineLocalMedia("wbot.t.1.message.send.media", cmd); err != nil {
			t.Fatal(err)
		}
		p := payloadOf(cmd)
		if p["mediaData"] != base64.StdEncoding.EncodeToString([]byte("PNGDATA")) {
			t.Fatalf("mediaData = %v", p["mediaData"])
		}
		if p["mediaUrl"] != url {
			t.Fatalf("mediaUrl precisa continuar (nome do arquivo em documentos): %v", p["mediaUrl"])
		}
	})

	t.Run("URL externa passa intacta, sem baixar nada", func(t *testing.T) {
		cmd := sendMediaCmd("https://cdn.exemplo.com/a.png")
		if err := inlineLocalMedia("wbot.t.1.message.send.media", cmd); err != nil {
			t.Fatal(err)
		}
		if _, ok := payloadOf(cmd)["mediaData"]; ok {
			t.Fatal("URL externa não deve virar mediaData")
		}
	})

	t.Run("arquivo local que sumiu falha ALTO, não vira relógio eterno", func(t *testing.T) {
		if err := inlineLocalMedia("wbot.t.1.message.send.media", sendMediaCmd("/public/media/sumiu.png")); err == nil {
			t.Fatal("esperava erro")
		}
	})

	t.Run("outros comandos não são tocados", func(t *testing.T) {
		cmd := map[string]interface{}{"type": "message.send.text", "payload": map[string]interface{}{"mediaUrl": url}}
		if err := inlineLocalMedia("wbot.t.1.message.send.text", cmd); err != nil {
			t.Fatal(err)
		}
		if _, ok := payloadOf(cmd)["mediaData"]; ok {
			t.Fatal("só message.send.media recebe bytes")
		}
	})

	t.Run("mediaData já informado (Assistant) não é sobrescrito", func(t *testing.T) {
		cmd := sendMediaCmd(url)
		payloadOf(cmd)["mediaData"] = "JA-TENHO"
		if err := inlineLocalMedia("wbot.t.1.message.send.media", cmd); err != nil {
			t.Fatal(err)
		}
		if payloadOf(cmd)["mediaData"] != "JA-TENHO" {
			t.Fatal("sobrescreveu mediaData")
		}
	})

	_ = os.Remove(filepath.Join(dir, url))
}
