package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callUpdate(h gin.HandlerFunc, method string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/x", h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, "/x", nil))
	return w
}

func fakeGitHub(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/xnetinho/watink/releases/latest", r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	old := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = old; srv.Close() })
}

func TestLatestRelease_LeTagEChangelogDoFork(t *testing.T) {
	t.Setenv("WATINK_RELEASES_REPO", "")
	fakeGitHub(t, 200, `{"tag_name":"v1.2.0","body":"## Novidades\n- RLS real\n* Telegram\ntexto solto"}`)

	w := callUpdate(GetLatestRelease, http.MethodGet)
	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "v1.2.0", got["version"])
	assert.Equal(t, []any{"RLS real", "Telegram"}, got["changelog"])
}

func TestLatestRelease_SemReleaseDegradaSemErro(t *testing.T) {
	t.Setenv("WATINK_RELEASES_REPO", "")
	fakeGitHub(t, 404, `{"message":"Not Found"}`)

	w := callUpdate(GetLatestRelease, http.MethodGet)
	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "-", got["version"])
}

func TestReleasesRepo_EnvInvalidoCaiNoPadrao(t *testing.T) {
	t.Setenv("WATINK_RELEASES_REPO", "../../etc/passwd")
	assert.Equal(t, defaultReleasesRepo, releasesRepo())
	t.Setenv("WATINK_RELEASES_REPO", "meu-org/meu-fork")
	assert.Equal(t, "meu-org/meu-fork", releasesRepo())
}

func TestStartUpdate_NaoDerrubaOServidor(t *testing.T) {
	w := callUpdate(StartUpdate, http.MethodPost)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "docker compose pull")
	mu.RLock()
	defer mu.RUnlock()
	assert.False(t, maintenanceMode, "não liga o modo manutenção")
}
