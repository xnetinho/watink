package services

import (
	"os"
	"strconv"
	"strings"
)

// defaultAutoDownloadMaxBytes: acima disso a mídia segue sob demanda (botão). Imagem, sticker, áudio e documento
// pequenos baixam sozinhos; vídeo e arquivo grande não, para não encher o disco nem ocupar o engine sem o operador pedir.
const defaultAutoDownloadMaxBytes int64 = 16 << 20

// autoDownloadTypes: o que baixa na chegada. Vídeo fica de fora de propósito (pesa e raramente é aberto).
var autoDownloadTypes = map[string]bool{"image": true, "sticker": true, "audio": true, "document": true}

// shouldAutoDownload decide se a mídia recebida baixa sozinha. maxBytes <= 0 desliga; tamanho desconhecido (0) não baixa.
func shouldAutoDownload(mediaType string, fileLength, maxBytes int64, hasProto bool) bool {
	if !hasProto || maxBytes <= 0 || !autoDownloadTypes[mediaType] {
		return false
	}
	return fileLength > 0 && fileLength <= maxBytes
}

// autoDownloadMaxBytes lê WATINK_MEDIA_AUTODOWNLOAD_MAX_BYTES ("0" desliga o recurso).
func autoDownloadMaxBytes() int64 {
	v := strings.TrimSpace(os.Getenv("WATINK_MEDIA_AUTODOWNLOAD_MAX_BYTES"))
	if v == "" {
		return defaultAutoDownloadMaxBytes
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultAutoDownloadMaxBytes
	}
	return n
}
