package groupsapi

import (
	"context"
	"log"
	"net/http"
	"os"
)

// Start launches the internal groups/communities/WhatsApp-version HTTP API and blocks until ctx is cancelled — same
// lifecycle shape as internal/health.Start, called alongside it from cmd/engine/main.go.
//
// There is no authentication: like the RabbitMQ commands and the calls audio channel (ADR 0031), the defence is the
// network. The port is exposed (never published with `ports:`) and only the other containers of the compose reach it.
func Start(ctx context.Context, backend Backend) {
	port := os.Getenv("GROUPS_API_PORT")
	if port == "" {
		port = "8084"
	}

	srv := &http.Server{Addr: ":" + port, Handler: newMux(backend, newThrottle())}

	go func() {
		log.Printf("[groupsapi] listening on :%s (docker-internal only)", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[groupsapi] server error: %v", err)
		}
	}()

	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
	log.Println("[groupsapi] server stopped")
}
