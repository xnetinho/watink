package enginego

import (
	"context"
	"net/http"
)

// WAVersionProbe é o resultado de consultar uma fonte de versão no engine.
type WAVersionProbe struct {
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WAVersionStatus espelha waversion.Status do engine: as versões lado a lado e a escolha vigente.
type WAVersionStatus struct {
	Mode      string         `json:"mode"`
	Manual    string         `json:"manual,omitempty"`
	Embedded  string         `json:"embedded"`
	Official  WAVersionProbe `json:"official"`
	Registry  WAVersionProbe `json:"registry"`
	Effective string         `json:"effective"`
	Source    string         `json:"source,omitempty"`
}

// GetWAVersionStatus pede ao engine as versões do cliente WhatsApp (embutida, oficial, registro wa-version, vigente).
func GetWAVersionStatus(ctx context.Context) (WAVersionStatus, error) {
	var out WAVersionStatus
	err := groupsDo(ctx, http.MethodGet, "/wa-version", nil, &out)
	return out, err
}

// RefreshWAVersion manda o engine reler a escolha do superadmin e aplicá-la nas próximas conexões.
func RefreshWAVersion(ctx context.Context) (WAVersionStatus, error) {
	var out struct {
		Status WAVersionStatus `json:"status"`
	}
	err := groupsDo(ctx, http.MethodPost, "/wa-version/refresh", nil, &out)
	return out.Status, err
}
