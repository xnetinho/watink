// Package engineaddr monta os endereços do engine a partir de uma única variável.
//
// ENGINE_HOST é só o nome (ou IP) do engine na rede interna, por exemplo "watink-engine".
// Daí saem as três portas internas, cada uma com o seu esquema. Cada endereço ainda pode
// ser sobreposto pela variável antiga (ENGINE_HEALTH_URL, GROUPS_API_URL, CALLS_AUDIO_URL),
// para instalações que já as definiram continuarem iguais.
package engineaddr

import (
	"os"
	"strings"
)

const (
	healthPort = "8083"
	groupsPort = "8084"
	callsPort  = "8085"
)

// Host devolve o ENGINE_HOST sem espaços, esquema nem porta. Vazio se não definido.
func Host() string {
	h := strings.TrimSpace(os.Getenv("ENGINE_HOST"))
	for _, p := range []string{"https://", "http://", "wss://", "ws://"} {
		h = strings.TrimPrefix(h, p)
	}
	return strings.TrimRight(h, "/")
}

func derive(override, scheme, port, path string) string {
	if v := strings.TrimSpace(os.Getenv(override)); v != "" {
		return v
	}
	h := Host()
	if h == "" {
		return ""
	}
	return scheme + "://" + h + ":" + port + path
}

// HealthURL é o /health do engine (ENGINE_HEALTH_URL sobrepõe).
func HealthURL() string { return derive("ENGINE_HEALTH_URL", "http", healthPort, "/health") }

// GroupsURL é a base da API interna de grupos (GROUPS_API_URL sobrepõe).
func GroupsURL() string { return derive("GROUPS_API_URL", "http", groupsPort, "") }

// CallsAudioURL é a base do WebSocket de áudio das chamadas (CALLS_AUDIO_URL sobrepõe).
func CallsAudioURL() string { return derive("CALLS_AUDIO_URL", "ws", callsPort, "") }
