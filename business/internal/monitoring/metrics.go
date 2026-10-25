package monitoring

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

// Métricas Prometheus para observabilidade do business (Tarefa 8.1).
// Implementação leve no formato padrão Prometheus (text/plain; version=0.0.4) sem dependências externas.

var (
	mu            sync.RWMutex
	messages      = make(map[string]*atomic.Int64)
	channelErrors = make(map[string]*atomic.Int64)
	queueDepth    = make(map[string]*atomic.Int64)
	activeCalls   atomic.Int64
)

func IncMessage(channel, direction, status string) {
	key := fmt.Sprintf(`channel="%s",direction="%s",status="%s"`, channel, direction, status)
	getCounter(messages, key).Add(1)
}

func IncChannelError(channel, errorType string) {
	key := fmt.Sprintf(`channel="%s",error_type="%s"`, channel, errorType)
	getCounter(channelErrors, key).Add(1)
}

func SetActiveCalls(n int64) {
	activeCalls.Store(n)
}

func SetQueueDepth(queue string, n int64) {
	key := fmt.Sprintf(`queue="%s"`, queue)
	getCounter(queueDepth, key).Store(n)
}

func getCounter(m map[string]*atomic.Int64, key string) *atomic.Int64 {
	mu.RLock()
	c, ok := m[key]
	mu.RUnlock()
	if ok {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if c, ok = m[key]; ok {
		return c
	}
	c = &atomic.Int64{}
	m[key] = c
	return c
}

// Handler expõe as métricas na rota /metrics no formato padrão do Prometheus.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		mu.RLock()
		defer mu.RUnlock()

		_, _ = fmt.Fprintln(w, "# HELP watink_messages_total Total de mensagens processadas por canal, direção e status")
		_, _ = fmt.Fprintln(w, "# TYPE watink_messages_total counter")
		for labels, count := range messages {
			_, _ = fmt.Fprintf(w, "watink_messages_total{%s} %d\n", labels, count.Load())
		}

		_, _ = fmt.Fprintln(w, "# HELP watink_channel_errors_total Total de erros por canal e tipo")
		_, _ = fmt.Fprintln(w, "# TYPE watink_channel_errors_total counter")
		for labels, count := range channelErrors {
			_, _ = fmt.Fprintf(w, "watink_channel_errors_total{%s} %d\n", labels, count.Load())
		}

		_, _ = fmt.Fprintln(w, "# HELP watink_active_calls Chamadas de voz ativas no momento")
		_, _ = fmt.Fprintln(w, "# TYPE watink_active_calls gauge")
		_, _ = fmt.Fprintf(w, "watink_active_calls %d\n", activeCalls.Load())

		_, _ = fmt.Fprintln(w, "# HELP watink_queue_depth Profundidade das filas de mensagens")
		_, _ = fmt.Fprintln(w, "# TYPE watink_queue_depth gauge")
		for labels, depth := range queueDepth {
			_, _ = fmt.Fprintf(w, "watink_queue_depth{%s} %d\n", labels, depth.Load())
		}
	})
}
