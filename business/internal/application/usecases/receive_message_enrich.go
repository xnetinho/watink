package usecases

import "strings"

// jidNumber extracts the phone number from a WhatsApp JID (e.g. "5511999@s.whatsapp.net" → "5511999").
func jidNumber(jid string) string {
	if jid == "" {
		return ""
	}
	base := strings.Split(jid, "@")[0]
	base = strings.Split(base, ":")[0]
	return base
}

// contactDisplayName returns the display name to use for the contact.
// For group messages the group subject takes precedence over the sender's push name.
func contactDisplayName(pushName, groupName string, isGroup bool) string {
	if isGroup && groupName != "" {
		return groupName
	}
	return pushName
}

// mimeTypeLabel returns a human-readable fallback label for a media message when
// no text body is present.
// viewOnceNotice é o texto do aviso de visualização única. O WhatsApp não entrega esse conteúdo a aparelhos
// vinculados, então o chat só registra que ela chegou e orienta a ver no celular.
const viewOnceNotice = "👁 Visualização única (o conteúdo só pode ser visto no celular)"

func mimeTypeLabel(mimetype string) string {
	switch {
	case strings.HasPrefix(mimetype, "image/"):
		return "📷 Foto"
	case strings.HasPrefix(mimetype, "video/"):
		return "📹 Vídeo"
	case strings.HasPrefix(mimetype, "audio/"):
		return "🎵 Áudio"
	case mimetype != "":
		return "📎 Arquivo"
	default:
		return ""
	}
}
