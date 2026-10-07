package calls

import "errors"

// Formato do quadro de VÍDEO no WebSocket de áudio (engine ↔ business ↔ navegador). O áudio PCM
// continua exatamente como antes: mensagem binária de 640 bytes, sem cabeçalho. O vídeo é uma mensagem
// binária que COMEÇA com o prefixo mágico, para o receptor distinguir sem mudar o áudio.
//
//	[FF 56 44 01][flags 1B][ts90k 4B BE][access unit Annex-B…]
//
// flags: bit0 = quadro-chave. (bits 1-2 reservados para a rotação, na fase de envio.)
var videoMagic = [4]byte{0xFF, 'V', 'D', 0x01}

const (
	videoHeaderLen = 4 + 1 + 4
	videoFlagKey   = 0x01
)

// ErrNotVideoFrame: a mensagem não é um quadro de vídeo (é PCM ou lixo).
var ErrNotVideoFrame = errors.New("não é um quadro de vídeo")

// EncodeVideoFrame monta a mensagem binária de um quadro de vídeo.
func EncodeVideoFrame(ts90k uint32, keyframe bool, accessUnit []byte) []byte {
	out := make([]byte, videoHeaderLen+len(accessUnit))
	copy(out, videoMagic[:])
	if keyframe {
		out[4] = videoFlagKey
	}
	out[5], out[6], out[7], out[8] = byte(ts90k>>24), byte(ts90k>>16), byte(ts90k>>8), byte(ts90k)
	copy(out[videoHeaderLen:], accessUnit)
	return out
}

// IsVideoFrame diz se a mensagem binária é um quadro de vídeo. PCM tem sempre múltiplo de 2 bytes e
// quase sempre 640; o prefixo de 4 bytes mais o tamanho mínimo do cabeçalho eliminam a ambiguidade na
// prática, e o chamador só testa isto em mensagens binárias.
func IsVideoFrame(msg []byte) bool {
	return len(msg) > videoHeaderLen && msg[0] == videoMagic[0] && msg[1] == videoMagic[1] &&
		msg[2] == videoMagic[2] && msg[3] == videoMagic[3]
}

// DecodeVideoFrame lê a mensagem; o slice devolvido aponta para dentro de msg.
func DecodeVideoFrame(msg []byte) (ts90k uint32, keyframe bool, accessUnit []byte, err error) {
	if !IsVideoFrame(msg) {
		return 0, false, nil, ErrNotVideoFrame
	}
	ts := uint32(msg[5])<<24 | uint32(msg[6])<<16 | uint32(msg[7])<<8 | uint32(msg[8])
	return ts, msg[4]&videoFlagKey != 0, msg[videoHeaderLen:], nil
}
