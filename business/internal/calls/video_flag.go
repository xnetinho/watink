package calls

import (
	"os"
	"strings"
)

// IsVideoEnabled indica se o suporte experimental a vídeo em chamadas está ativo.
// Desligado por padrão para proteção de estabilidade e conformidade.
func IsVideoEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_EXPERIMENTAL_VIDEO_CALLS")))
	return v == "true" || v == "1" || v == "yes"
}
