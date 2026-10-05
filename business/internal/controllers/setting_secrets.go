package controllers

import (
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/models"
)

// maskedSecretPlaceholder substitui o valor real de uma setting secreta para
// quem não tem settings:update. O frontend usa o prefixo para saber que há um
// valor configurado sem conhecê-lo.
const maskedSecretPlaceholder = "••••••••"

// secretSettingSuffixes: qualquer chave que termine assim é tratada como
// segredo, inclusive chaves futuras (aiEmbeddingApiKey, integrationToken...).
// Casamento case-insensitive no sufixo.
var secretSettingSuffixes = []string{"apikey", "secret", "token", "password", "passwd"}

// isSecretSettingKey diz se o valor da setting nunca deve ser exibido a quem
// só tem permissão de leitura.
func isSecretSettingKey(key string) bool {
	k := strings.ToLower(key)
	for _, suf := range secretSettingSuffixes {
		if strings.HasSuffix(k, suf) {
			return true
		}
	}
	return false
}

// maskSecretSettings devolve uma cópia da lista com os valores secretos
// substituídos por um placeholder (e só quando há valor: setting vazia continua
// vazia, para a UI distinguir "não configurado" de "configurado").
func maskSecretSettings(in []models.Setting) []models.Setting {
	out := make([]models.Setting, len(in))
	copy(out, in)
	for i := range out {
		if isSecretSettingKey(out[i].Key) && out[i].Value != "" {
			out[i].Value = maskedSecretPlaceholder
		}
	}
	return out
}
