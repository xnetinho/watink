package health

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler_Returns200WithJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(`{"status":"ok","service":"watink-engine"}`))
	})
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("expected status:ok in body, got %q", body)
	}
	if !strings.Contains(body, `"service":"watink-engine"`) {
		t.Fatalf("expected service:watink-engine in body, got %q", body)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func TestHandler_ReportsCallsLoad(t *testing.T) {
	h := Handler(func() (int, int) { return 3, 17 })
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, `"active":3`) || !strings.Contains(body, `"audioQueued":17`) || !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("code=%d body=%s", w.Code, body)
	}
}

func TestHandler_WithoutLoadOmitsCalls(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if strings.Contains(w.Body.String(), "calls") || !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("body=%s", w.Body.String())
	}
}
