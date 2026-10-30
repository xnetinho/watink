package whatsapp

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
)

// buildMediaMessage monta a mensagem de mídia. Com ViewOnce, o cliente oficial manda a mídia dentro do envelope
// ViewOnceMessageV2 (além do campo ViewOnce da própria mídia): é o envelope que faz o aparelho do destinatário tratá-la como
// visualização única. Documento e figurinha não têm esse modo: o pedido é ignorado.
func buildMediaMessage(payload MediaCommandPayload, uploaded whatsmeow.UploadResponse, meta mediameta.Meta) *waE2E.Message {
	msg := buildMediaPayload(payload, uploaded, meta)
	if payload.ViewOnce && wrapsViewOnce(msg) {
		return &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: msg}}
	}
	return msg
}

// wrapsViewOnce diz se a mídia montada aceita visualização única (foto, vídeo ou áudio).
func wrapsViewOnce(m *waE2E.Message) bool {
	return m.GetImageMessage() != nil || m.GetVideoMessage() != nil || m.GetAudioMessage() != nil
}
