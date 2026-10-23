// Package engineaddr monta os endereços do engine a partir de uma única variável.
//
// ENGINE_HOST é só o nome (ou IP) do engine na rede interna, por exemplo "watink-engine".
// Daí saem as três portas internas, cada uma com o seu esquema: /health (http, 8083), API de
// grupos (http, 8084) e áudio das chamadas (ws, 8085).
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

func build(scheme, port, path string) string {
	h := Host()
	if h == "" {
		return ""
	}
	return scheme + "://" + h + ":" + port + path
}

// HealthURL é o /health do engine. Vazio sem ENGINE_HOST.
func HealthURL() string { return build("http", healthPort, "/health") }

// GroupsURL é a base da API interna de grupos. Vazio sem ENGINE_HOST.
func GroupsURL() string { return build("http", groupsPort, "") }

// CallsAudioURL é a base do WebSocket de áudio das chamadas. Vazio sem ENGINE_HOST.
func CallsAudioURL() string { return build("ws", callsPort, "") }
