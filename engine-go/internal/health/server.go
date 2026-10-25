package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Load informa a carga de chamadas: chamadas ativas e quadros de áudio na fila.
type Load func() (activeCalls int, audioQueued int)

// Handler monta o /health e /metrics (Tarefa 8.1). load pode ser nil (sem chamadas a reportar).
func Handler(load Load) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		body := map[string]interface{}{"status": "ok", "service": "watink-engine"}
		if load != nil {
			active, queued := load()
			body["calls"] = map[string]int{"active": active, "audioQueued": queued}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		active, queued := 0, 0
		if load != nil {
			active, queued = load()
		}

		_, _ = fmt.Fprintln(w, "# HELP watink_engine_active_calls Chamadas de voz ativas no engine")
		_, _ = fmt.Fprintln(w, "# TYPE watink_engine_active_calls gauge")
		_, _ = fmt.Fprintf(w, "watink_engine_active_calls %d\n", active)

		_, _ = fmt.Fprintln(w, "# HELP watink_engine_audio_queued Quadros de audio na fila do engine")
		_, _ = fmt.Fprintln(w, "# TYPE watink_engine_audio_queued gauge")
		_, _ = fmt.Fprintf(w, "watink_engine_audio_queued %d\n", queued)
	})

	return mux
}

// Start launches a minimal HTTP health server on the configured port (default 8083).
// It blocks until ctx is cancelled, then shuts down gracefully.
func Start(ctx context.Context, load Load) {
	port := os.Getenv("HEALTH_PORT")
	if port == "" {
		port = "8083"
	}

	mux := Handler(load)

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("Health server listening on :%s/health", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()

	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
	log.Println("Health server stopped")
}
