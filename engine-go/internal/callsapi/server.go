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

// maxClientMessage cobre um quadro-chave de câmera (o PCM tem 640 B): 512 KB, o mesmo teto do business.
const maxClientMessage = 512 * 1024

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
	c.SetReadLimit(maxClientMessage)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer cancel()
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			switch typ {
			case websocket.MessageBinary:
				if calls.IsVideoFrame(data) {
					pipe.WriteVideo(data)
				} else {
					pipe.WritePCM(data)
				}
			case websocket.MessageText:
				handleCommand(ctx, pipe, data)
			}
		}
	}()

	var videoTs uint32
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
		case v := <-pipe.Video():
			// O relógio de 90 kHz do WebCodecs é só uma referência de apresentação: o decoder do navegador
			// usa o timestamp que vier. Um contador por quadro (≈15 fps) basta para a fase de recepção.
			videoTs += 6000
			if !write(ctx, c, websocket.MessageBinary, calls.EncodeVideoFrame(videoTs, v.Keyframe, v.Rotation, v.AccessUnit)) {
				return
			}
		case ctl := <-pipe.Control():
			raw, _ := json.Marshal(ctl)
			if !write(ctx, c, websocket.MessageText, raw) {
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

// command é o comando de texto que o business manda no mesmo WebSocket do áudio.
type command struct {
	Type        string `json:"type"`
	On          bool   `json:"on"`
	Orientation int    `json:"orientation"`
}

// handleCommand trata um comando de controle do business. Hoje só "camera" (liga/desliga com a
// orientação); qualquer outra coisa é ignorada.
func handleCommand(ctx context.Context, pipe *calls.AudioPipe, raw []byte) {
	var c command
	if json.Unmarshal(raw, &c) != nil || c.Type != "camera" {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pipe.SetCamera(cctx, c.On, c.Orientation); err != nil {
		log.Printf("[callsapi] camera on=%v: %v", c.On, err)
	}
}

func write(ctx context.Context, c *websocket.Conn, typ websocket.MessageType, b []byte) bool {
	wctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Write(wctx, typ, b) == nil
}
