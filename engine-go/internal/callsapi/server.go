// Package callsapi expõe o ÚNICO endpoint direto engine↔business: o WebSocket
// interno de áudio de uma chamada. Só docker-internal (expose, nunca ports): sem
// autenticação própria, quem alcança a porta alcança o áudio. O resto da comunicação
// é RabbitMQ.
package callsapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/calls"
	"github.com/coder/websocket"
)

var callIDPattern = regexp.MustCompile(`^[A-F0-9]{32}$`)

// Backend é o que o servidor precisa do engine.
type Backend interface {
	OpenCallAudio(callID string) (*calls.AudioPipe, error)
	CallsLoad() (active int, audioQueued int)
}

// Start sobe o servidor e só retorna quando ctx é cancelado.
func Start(ctx context.Context, b Backend) {
	port := os.Getenv("CALLS_AUDIO_PORT")
	if port == "" {
		port = "8085"
	}
	srv := &http.Server{Addr: ":" + port, Handler: NewHandler(b)}
	go func() {
		log.Printf("[callsapi] listening on :%s (docker-internal only)", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[callsapi] server error: %v", err)
		}
	}()
	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
}

// NewHandler monta o roteador do canal de áudio.
func NewHandler(b Backend) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /calls/{id}/audio", func(w http.ResponseWriter, r *http.Request) {
		serveAudio(w, r, b)
	})
	return mux
}

func serveAudio(w http.ResponseWriter, r *http.Request, b Backend) {
	id := r.PathValue("id")
	if !callIDPattern.MatchString(id) {
		http.Error(w, `{"error":"invalid call id"}`, http.StatusBadRequest)
		return
	}
	pipe, err := b.OpenCallAudio(id)
	switch {
	case errors.Is(err, calls.ErrNoCall):
		http.Error(w, `{"error":"call not found"}`, http.StatusNotFound)
		return
	case errors.Is(err, calls.ErrAudioInUse):
		http.Error(w, `{"error":"audio already open"}`, http.StatusConflict)
		return
	case err != nil:
		http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	defer pipe.Close()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Só o business (processo servidor, sem navegador nem Origin) conecta aqui;
		// a checagem de Origin não se aplica.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(64 * 1024)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer cancel()
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				pipe.WritePCM(data)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			_ = c.Close(websocket.StatusNormalClosure, "")
			return
		case <-pipe.Done():
			_ = c.Close(websocket.StatusNormalClosure, "call ended")
			return
		case f := <-pipe.Out():
			if !write(ctx, c, websocket.MessageBinary, f) {
				return
			}
		case t := <-pipe.Telemetry():
			raw, _ := json.Marshal(t)
			if !write(ctx, c, websocket.MessageText, raw) {
				return
			}
		}
	}
}

func write(ctx context.Context, c *websocket.Conn, typ websocket.MessageType, b []byte) bool {
	wctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Write(wctx, typ, b) == nil
}
