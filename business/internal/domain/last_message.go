package domain

import "strings"

// LastMessagePreview decide o que a lista de tickets mostra como "última mensagem": o TEXTO real, quando existe, ou um
// MARCADOR de tipo (image, video, poll...) que a tela traduz. O banco nunca guarda rótulo em um idioma, então o mesmo
// ticket aparece em pt, en ou es conforme o idioma de quem olha.
//
//	texto legível            -> (texto, "")
//	mídia/cartão sem legenda -> ("", tipo)
//	tipo com corpo opaco     -> ("", tipo)  (coordenadas, vCard, nome de campo, enquete formatada...)
func LastMessagePreview(mediaType, body, mimeType string) (text, kind string) {
	t := strings.TrimSpace(mediaType)
	if opaqueBodyTypes[t] {
		return "", t
	}
	if strings.TrimSpace(body) != "" {
		return body, ""
	}
	if t != "" && t != "chat" {
		return "", t
	}
	return "", kindFromMime(mimeType)
}

// opaqueBodyTypes: o body desses tipos não é texto para o operador (coordenadas, vCard, nome de campo, texto formatado de
// enquete/evento): a prévia é sempre o marcador.
var opaqueBodyTypes = map[string]bool{"view_once": true, "location": true, "vcard": true, "unsupported": true, "poll": true, "event": true, "pix": true}

// kindFromMime marca mídia que chegou sem tipo explícito, a partir do mimetype.
func kindFromMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	case mime != "":
		return "document"
	}
	return ""
}
