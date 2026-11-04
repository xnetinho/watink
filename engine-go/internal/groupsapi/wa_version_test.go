package groupsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/waversion"
)

// fakeVersions troca só os dois métodos de versão do fakeBackend.
type fakeVersions struct {
	fakeBackend
	status  waversion.Status
	err     error
	changed bool
	forced  bool
}

func (f *fakeVersions) WAVersionStatus(context.Context) (waversion.Status, error) {
	return f.status, f.err
}

func (f *fakeVersions) RefreshWAVersion(_ context.Context, force bool) bool {
	f.forced = force
	return f.changed
}

func call(t *testing.T, b Backend, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	newMux(b, newThrottle()).ServeHTTP(rec, req)
	return rec
}

func TestWAVersionStatus(t *testing.T) {
	f := &fakeVersions{status: waversion.Status{Mode: "manual", Manual: "2.3000.5", Embedded: "2.3000.1", Effective: "2.3000.5"}}
	rec := call(t, f, http.MethodGet, "/wa-version")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var env struct {
		OK   bool             `json:"ok"`
		Data waversion.Status `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || !env.OK || env.Data.Mode != "manual" || env.Data.Effective != "2.3000.5" {
		t.Fatalf("resposta inesperada: %v %s", err, rec.Body)
	}
}

func TestWAVersionRefreshForcaEDevolveStatus(t *testing.T) {
	f := &fakeVersions{changed: true, status: waversion.Status{Effective: "2.3000.9"}}
	rec := call(t, f, http.MethodPost, "/wa-version/refresh")
	if rec.Code != http.StatusOK || !f.forced {
		t.Fatalf("status %d forced=%v", rec.Code, f.forced)
	}
	var env struct {
		Data struct {
			Changed bool             `json:"changed"`
			Status  waversion.Status `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || !env.Data.Changed || env.Data.Status.Effective != "2.3000.9" {
		t.Fatalf("resposta inesperada: %v %s", err, rec.Body)
	}
}

func TestWAVersionErroDoBackend(t *testing.T) {
	rec := call(t, &fakeVersions{err: errors.New("fora")}, http.MethodGet, "/wa-version")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", rec.Code)
	}
}
