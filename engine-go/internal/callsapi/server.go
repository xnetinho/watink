// Package callsapi expõe o ÚNICO endpoint direto engine↔business: o WebSocket
// interno de áudio de uma chamada. Só docker-internal (expose, nunca ports) e
// fail-closed sem CALLS_AUDIO_TOKEN. O resto da comunicação é RabbitMQ.
package callsapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
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

// Token devolve o token configurado; vazio significa servidor desligado.
func Token() string { return strings.TrimSpace(os.Getenv("CALLS_AUDIO_TOKEN")) }

// Start sobe o servidor. Sem CALLS_AUDIO_TOKEN NÃO sobe (fail-closed).
func Start(ctx context.Context, b Backend) {
	token := Token()
	if token == "" {
		log.Println("[callsapi] CALLS_AUDIO_TOKEN not set — internal call audio API disabled (fail-closed)")
		return
	}
	port := os.Getenv("CALLS_AUDIO_PORT")
	if port == "" {
		port = "8085"
	}
	srv := &http.Server{Addr: ":" + port, Handler: NewHandler(b, token)}
	go func() {
		log.Printf("[callsapi] listening on :%s (docker-internal only)", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[callsapi] server error: %v", err)
		}
	}()
	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
}

// NewHandler monta o roteador autenticado por X-Internal-Token.
func NewHandler(b Backend, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /calls/{id}/audio", func(w http.ResponseWriter, r *http.Request) {
		serveAudio(w, r, b)
	})
	return auth(token, mux)
}

func auth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Internal-Token")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, `{"error":"missing or invalid X-Internal-Token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
		// Só o business (processo servidor, sem navegador nem Origin) conecta aqui, já
		// autenticado por X-Internal-Token; a checagem de Origin não se aplica.
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
