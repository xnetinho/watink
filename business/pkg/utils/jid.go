package utils

import (
	"strings"
)

// ContactJID constrói o endereço canônico de envio no WhatsApp para um contato.
// Grupos usam @g.us. Contatos com LID usam o LID direto. Demais usam o número.
// Centraliza a lógica de endereçamento antes espalhada em 6+ pontos (Tarefa 4.3 / ADR 0033).
func ContactJID(number string, lid *string, isGroup bool) string {
	if isGroup {
		if strings.HasSuffix(number, "@g.us") {
			return number
		}
		return number + "@g.us"
	}
	if lid != nil && *lid != "" {
		return *lid
	}
	return number
}

// UserJID constrói o JID completo de um usuário WhatsApp ("<number>@s.whatsapp.net").
func UserJID(number string) string {
	clean := strings.TrimSpace(number)
	if strings.Contains(clean, "@") {
		return clean
	}
	return clean + "@s.whatsapp.net"
}

// ExtractNumberFromJID extrai apenas os dígitos ou identificador limpo de um JID.
func ExtractNumberFromJID(jid string) string {
	clean := strings.TrimSpace(jid)
	if idx := strings.Index(clean, "@"); idx != -1 {
		clean = clean[:idx]
	}
	if idx := strings.Index(clean, ":"); idx != -1 {
		clean = clean[:idx]
	}
	return clean
}
