package telegram

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

// PollingManager mantém um worker de long-polling por conexão Telegram.
type PollingManager struct {
	mu      sync.Mutex
	workers map[int]context.CancelFunc
}

var globalPollingManager = &PollingManager{
	workers: make(map[int]context.CancelFunc),
}

func (pm *PollingManager) Start(conn *models.ChannelConnection, handler InboundHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Para worker anterior se houver
	if cancel, exists := pm.workers[conn.ID]; exists {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	pm.workers[conn.ID] = cancel

	go func() {
		token, err := domain.ChannelCredentials(conn)
		if err != nil {
			log.Printf("[Telegram Polling] connection %d has no token: %v", conn.ID, err)
			return
		}
		client := NewClient(token)
		// Offset em memória: após restart o Telegram devolve updates pendentes e a
		// deduplicação por externalMessageId no serviço de entrada descarta repetidos.
		var offset int64 = 0

		log.Printf("[Telegram Polling] Started polling worker for connection %d (%s)", conn.ID, conn.Name)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[Telegram Polling] Stopped polling worker for connection %d", conn.ID)
				return
			default:
				updates, err := client.GetUpdates(ctx, offset, 20)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("[Telegram Polling] error fetching updates (conn %d): %v", conn.ID, err)
					time.Sleep(3 * time.Second)
					continue
				}

				adapter := NewAdapter(handler)
				for _, update := range updates {
					offset = update.UpdateID + 1
					raw, _ := json.Marshal(update)
					msgs, parseErr := adapter.ParseInbound(ctx, conn, raw, nil)
					if parseErr != nil {
						log.Printf("[Telegram Polling] parse error (conn %d): %v", conn.ID, parseErr)
						continue
					}
					if handler == nil {
						continue
					}
					for _, msg := range msgs {
						if rxErr := handler(ctx, msg); rxErr != nil {
							log.Printf("[Telegram Polling] receive error (conn %d): %v", conn.ID, rxErr)
						}
					}
				}
			}
		}
	}()
}

func (pm *PollingManager) Stop(connID int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if cancel, exists := pm.workers[connID]; exists {
		cancel()
		delete(pm.workers, connID)
	}
}

// ResumePolling reinicia os workers das conexões Telegram configuradas em polling,
// para que o modo escolhido sobreviva a restart do processo.
func ResumePolling(conns []models.ChannelConnection, handler InboundHandler) {
	for i := range conns {
		conn := conns[i]
		globalPollingManager.Start(&conn, handler)
	}
}
