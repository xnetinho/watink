package groupsapi

import (
	"context"
	"net/http"
	"time"
)

// handleWAVersionStatus devolve as versões do cliente WhatsApp lado a lado (embutida, oficial, registro wa-version e a
// vigente) para a tela do superadmin. Consulta as fontes na hora, por isso pode levar alguns segundos.
func handleWAVersionStatus(backend Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		st, err := backend.WAVersionStatus(ctx)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, CodeProviderError, err.Error())
			return
		}
		writeData(w, http.StatusOK, st)
	}
}

// handleWAVersionRefresh relê a escolha do superadmin e aplica a versão. Sessões conectadas não caem; vale na próxima conexão.
func handleWAVersionRefresh(backend Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		changed := backend.RefreshWAVersion(ctx, true)
		st, err := backend.WAVersionStatus(ctx)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, CodeProviderError, err.Error())
			return
		}
		writeData(w, http.StatusOK, map[string]any{"changed": changed, "status": st})
	}
}
