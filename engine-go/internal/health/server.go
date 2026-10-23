package health

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// Load informa a carga de chamadas: chamadas ativas e quadros de áudio na fila.
type Load func() (activeCalls int, audioQueued int)

// Handler monta o /health. load pode ser nil (sem chamadas a reportar).
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
