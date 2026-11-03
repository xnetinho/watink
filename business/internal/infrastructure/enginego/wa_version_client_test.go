package enginego

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWAVersionStatus(t *testing.T) {
	setupGroupsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/wa-version", r.URL.Path)
		assert.Equal(t, "test-internal-token", r.Header.Get("X-Internal-Token"))
		_, _ = w.Write([]byte(`{"ok":true,"data":{"mode":"auto","embedded":"2.3000.1","official":{"version":"2.3000.3"},"registry":{"error":"sem rede"},"effective":"2.3000.3","source":"oficial"}}`))
	})
	st, err := GetWAVersionStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2.3000.3", st.Official.Version)
	assert.Equal(t, "sem rede", st.Registry.Error)
	assert.Equal(t, "oficial", st.Source)
}

func TestRefreshWAVersion(t *testing.T) {
	setupGroupsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/wa-version/refresh", r.URL.Path)
		_, _ = w.Write([]byte(`{"ok":true,"data":{"changed":true,"status":{"mode":"manual","effective":"2.3000.9"}}}`))
	})
	st, err := RefreshWAVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "manual", st.Mode)
	assert.Equal(t, "2.3000.9", st.Effective)
}
