package monitoring_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMetrics_ExposesSeries(t *testing.T) {
	monitoring.IncMessage("whatsapp", "inbound", "success")
	monitoring.IncChannelError("telegram", "rate_limit")
	monitoring.SetActiveCalls(3)
	monitoring.SetQueueDepth("engine.go.commands", 12)

	handler := monitoring.Handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, "watink_messages_total")
	assert.Contains(t, body, "watink_channel_errors_total")
	assert.Contains(t, body, "watink_active_calls 3")
	assert.Contains(t, body, "watink_queue_depth")
	assert.Contains(t, body, `channel="whatsapp"`)
	assert.Contains(t, body, `channel="telegram"`)
}
