package domain

import (
	"fmt"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/cryptobox"
)

// ChannelCredentials devolve o segredo do canal em texto plano para as chamadas à API externa.
// O valor é cifrado at-rest com cryptobox; conexões gravadas antes da cifragem seguem em texto
// plano e continuam funcionando (o decrypt falha e o valor é usado como está).
func ChannelCredentials(conn *models.ChannelConnection) (string, error) {
	raw := strings.TrimSpace(conn.CredentialsEnc)
	if raw == "" {
		return "", fmt.Errorf("channel connection %d has no credentials", conn.ID)
	}
	plain, err := cryptobox.Decrypt(raw)
	if err != nil {
		return raw, nil
	}
	if plain == "" {
		return raw, nil
	}
	return plain, nil
}
