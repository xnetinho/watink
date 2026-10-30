package whatsapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var voChat = types.NewJID("5511999990001", types.DefaultUserServer)

func voImage(complete bool) *waE2E.ImageMessage {
	img := &waE2E.ImageMessage{Mimetype: proto.String("image/jpeg"), FileLength: proto.Uint64(48213), ViewOnce: proto.Bool(true), JPEGThumbnail: []byte("THUMB-SECRETA")}
	if complete {
		img.URL = proto.String("https://mmg.whatsapp.net/SEGREDO-URL")
		img.DirectPath = proto.String("/v/SEGREDO-PATH")
		img.MediaKey = []byte("CHAVE-SECRETA-DA-MIDIA-XXXXXXXXXX")
		img.FileSHA256 = []byte("HASH-SECRETO-1")
		img.FileEncSHA256 = []byte("HASH-SECRETO-2")
	}
	return img
}

func voEvent(raw *waE2E.Message) *events.Message {
	evt := &events.Message{
		Info:       types.MessageInfo{ID: "VO1", Timestamp: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC), MessageSource: types.MessageSource{Chat: voChat, Sender: voChat}},
		RawMessage: raw,
	}
	return evt.UnwrapRaw()
}

func TestInspectViewOnce_MidiaCompletaDizQueDaParaBaixar(t *testing.T) {
	out := inspectViewOnceMessage(voEvent(&waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: &waE2E.Message{ImageMessage: voImage(true)}}}))
	for _, want := range []string{"id=VO1", "envelope=true", "isViewOnce=true", "imagem(", "bytes=48213", "url=sim", "directPath=sim", "mediaKey=sim", "sha256=sim", "encSha256=sim"} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q em: %s", want, out)
		}
	}
}

func TestInspectViewOnce_MidiaIncompletaDizOQueFalta(t *testing.T) {
	out := inspectViewOnceMessage(voEvent(&waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: &waE2E.Message{ImageMessage: voImage(false)}}}))
	for _, want := range []string{"url=nao", "directPath=nao", "mediaKey=nao"} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q em: %s", want, out)
		}
	}
}

// O registro existe para decidir a viabilidade, não para vazar a mídia: nenhum valor sensível pode aparecer.
func TestInspectViewOnce_NuncaExpoeChaveUrlHashNemMiniatura(t *testing.T) {
	cases := map[string]string{
		"mensagem": inspectViewOnceMessage(voEvent(&waE2E.Message{ViewOnceMessage: &waE2E.FutureProofMessage{Message: &waE2E.Message{ImageMessage: voImage(true)}}})),
		"citação": inspectViewOnceContext(&waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String("resposta"),
			ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("ALVO"), QuotedMessage: &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: &waE2E.Message{ImageMessage: voImage(true)}}}},
		}}),
	}
	for name, out := range cases {
		if out == "" {
			t.Fatalf("%s: esperava descrição", name)
		}
		for _, secret := range []string{"SEGREDO-URL", "SEGREDO-PATH", "CHAVE-SECRETA", "HASH-SECRETO", "THUMB-SECRETA", "mmg.whatsapp.net"} {
			if strings.Contains(out, secret) {
				t.Errorf("%s vazou %q: %s", name, secret, out)
			}
		}
	}
}

func TestInspectViewOnce_MensagemComumNaoGeraRegistro(t *testing.T) {
	if got := inspectViewOnceMessage(voEvent(&waE2E.Message{Conversation: proto.String("oi")})); got != "" {
		t.Errorf("mensagem comum não é visualização única: %s", got)
	}
	if got := inspectViewOnceMessage(voEvent(&waE2E.Message{ImageMessage: &waE2E.ImageMessage{Mimetype: proto.String("image/jpeg")}})); got != "" {
		t.Errorf("foto comum não é visualização única: %s", got)
	}
}

func TestInspectViewOnce_UnavailableDescreOAviso(t *testing.T) {
	v := &events.UndecryptableMessage{
		Info:            types.MessageInfo{ID: "U1", Timestamp: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC), MessageSource: types.MessageSource{Chat: voChat, Sender: voChat}},
		IsUnavailable:   true,
		UnavailableType: events.UnavailableTypeViewOnce,
	}
	out := inspectViewOnceUnavailable(v)
	for _, want := range []string{"UNAVAILABLE", "id=U1", "unavailable=true", `tipo="view_once"`} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q em: %s", want, out)
		}
	}
}

// A citação comum (resposta a uma mensagem normal) não pode poluir o diagnóstico.
func TestInspectViewOnce_CitacaoComumNaoGeraRegistro(t *testing.T) {
	got := inspectViewOnceContext(&waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String("ok"),
		ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("A"), QuotedMessage: &waE2E.Message{Conversation: proto.String("antes")}},
	}})
	if got != "" {
		t.Errorf("citação comum não é visualização única: %s", got)
	}
}

func TestViewOnceInspectEnabled(t *testing.T) {
	t.Setenv("WATINK_VIEWONCE_INSPECT", "")
	if viewOnceInspectEnabled() {
		t.Error("desligado por padrão")
	}
	for _, v := range []string{"1", "true", " TRUE "} {
		t.Setenv("WATINK_VIEWONCE_INSPECT", v)
		if !viewOnceInspectEnabled() {
			t.Errorf("%q deveria ligar", v)
		}
	}
}

// ---- recuperação da visualização única citada ----

type fakeDL struct {
	data []byte
	err  error
	got  whatsmeow.DownloadableMessage
}

func (f *fakeDL) Download(_ context.Context, m whatsmeow.DownloadableMessage) ([]byte, error) {
	f.got = m
	return f.data, f.err
}

func quotedViewOnce(img *waE2E.ImageMessage) *waE2E.Message {
	return &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String("Recebido"),
		ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("ALVO1"), QuotedMessage: &waE2E.Message{ImageMessage: img}},
	}}
}

func TestQuotedViewOnceMedia_AchaAMidiaCitadaDeVisualizacaoUnica(t *testing.T) {
	media, target, kind := quotedViewOnceMedia(quotedViewOnce(voImage(true)))
	if media == nil || target != "ALVO1" || kind != "image" {
		t.Fatalf("media=%v target=%q kind=%q", media, target, kind)
	}
}

// Resposta a uma foto COMUM não pode disparar download.
func TestQuotedViewOnceMedia_FotoComumNaoEAlvo(t *testing.T) {
	comum := voImage(true)
	comum.ViewOnce = proto.Bool(false)
	if media, _, _ := quotedViewOnceMedia(quotedViewOnce(comum)); media != nil {
		t.Fatal("foto comum citada não é visualização única")
	}
	if media, _, _ := quotedViewOnceMedia(&waE2E.Message{Conversation: proto.String("oi")}); media != nil {
		t.Fatal("mensagem sem citação não tem alvo")
	}
}

func TestRecoverViewOnce_PublicaAMidiaSobreOAviso(t *testing.T) {
	svc, calls := newTestService()
	img := voImage(true) // 48213 bytes declarados
	data := make([]byte, 48213)
	for i := range data {
		data[i] = byte(i)
	}
	svc.recoverViewOnce(context.Background(), &fakeDL{data: data}, "tenant-x", 1, img, "ALVO1", "image")

	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento, veio %d", len(*calls))
	}
	c := (*calls)[0]
	if c.eventType != "message.media" {
		t.Fatalf("evento = %q", c.eventType)
	}
	p := c.payload
	if p["messageId"] != "ALVO1" || p["mediaType"] != "image" || p["viewOnce"] != true || p["mimetype"] != "image/jpeg" || p["error"] != "" {
		t.Errorf("payload = %+v", p)
	}
	raw, err := base64.StdEncoding.DecodeString(p["mediaData"].(string))
	if err != nil || len(raw) != 48213 {
		t.Errorf("mediaData: %d bytes, err=%v", len(raw), err)
	}
}

// Só publica o que baixou ÍNTEGRO. Falha de rede, hash inválido ou tamanho diferente deixam o aviso como estava.
func TestRecoverViewOnce_NaoPublicaSeFalhouOuNaoConfere(t *testing.T) {
	cases := map[string]*fakeDL{
		"403":               {err: whatsmeow.ErrMediaDownloadFailedWith403},
		"404":               {err: whatsmeow.ErrMediaDownloadFailedWith404},
		"hash inválido":     {err: whatsmeow.ErrInvalidMediaSHA256},
		"rede":              {err: errors.New("rede caiu")},
		"tamanho diferente": {data: make([]byte, 100)},
		"vazio":             {data: nil},
	}
	for name, dl := range cases {
		t.Run(name, func(t *testing.T) {
			svc, calls := newTestService()
			svc.recoverViewOnce(context.Background(), dl, "t", 1, voImage(true), "ALVO1", "image")
			if len(*calls) != 0 {
				t.Fatalf("não podia publicar nada, veio %+v", *calls)
			}
		})
	}
}

// O arquivo e as chaves nunca vão para o log do engine.
func TestRecoverViewOnce_NaoVazaConteudoNoLog(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	svc, _ := newTestService()
	img := voImage(true)
	data := bytes.Repeat([]byte("CONTEUDO-SECRETO"), 48213/16+1)[:48213]
	svc.recoverViewOnce(context.Background(), &fakeDL{data: data}, "t", 1, img, "ALVO1", "image")
	svc.recoverViewOnce(context.Background(), &fakeDL{err: errors.New("falhou")}, "t", 1, img, "ALVO2", "image")
	for _, s := range []string{"CONTEUDO-SECRETO", "CHAVE-SECRETA", "SEGREDO-URL", "SEGREDO-PATH", "HASH-SECRETO"} {
		if strings.Contains(buf.String(), s) {
			t.Errorf("o log vazou %q: %s", s, buf.String())
		}
	}
}

// Ponta a ponta no engine: a resposta do celular principal (com a citação) dispara a recuperação.
func TestHandleMessageEvent_CitacaoDeVisualizacaoUnicaRecuperaAMidia(t *testing.T) {
	svc, calls := newTestService()
	img := voImage(true)
	evt := voEvent(quotedViewOnce(img))
	evt.Info.ID = "RESPOSTA1"
	evt.Info.IsFromMe = true

	media, target, kind := quotedViewOnceMedia(evt.Message)
	if media == nil {
		t.Fatal("a resposta deveria apontar para a visualização única")
	}
	svc.recoverViewOnce(context.Background(), &fakeDL{data: make([]byte, 48213)}, "t", 1, media, target, kind)
	found := false
	for _, c := range *calls {
		if c.eventType == "message.media" && c.payload["messageId"] == "ALVO1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("a mídia deveria ser publicada sobre o aviso ALVO1: %+v", *calls)
	}
}
