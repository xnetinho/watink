package telegram

import (
	"html"
	"regexp"
)

// O painel escreve na sintaxe do WhatsApp (*negrito*, _itálico_, ~riscado~, inclusive a assinatura "*Nome:*"); o Telegram só a entende como HTML
// (parse_mode). O texto é escapado antes, então "<" e "&" do atendente nunca viram tag.
// shortcut: sem bloco de código (```) nem ligação de marcadores entre linhas; adicionar se o atendente pedir.
var (
	waBold   = regexp.MustCompile(`(^|[^\w*])\*([^\s*](?:[^*\n]*[^\s*])?)\*`)
	waItalic = regexp.MustCompile(`(^|[^\w_])_([^\s_](?:[^_\n]*[^\s_])?)_`)
	waStrike = regexp.MustCompile(`(^|[^\w~])~([^\s~](?:[^~\n]*[^\s~])?)~`)
)

func whatsAppToTelegramHTML(s string) string {
	s = html.EscapeString(s)
	s = waBold.ReplaceAllString(s, "${1}<b>${2}</b>")
	s = waItalic.ReplaceAllString(s, "${1}<i>${2}</i>")
	return waStrike.ReplaceAllString(s, "${1}<s>${2}</s>")
}
