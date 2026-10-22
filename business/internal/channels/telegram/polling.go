package telegram

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
)

type PollingManager struct {
	mu      sync.Mutex
	workers map[int]context.CancelFunc
}

var globalPollingManager = &PollingManager{
	workers: make(map[int]context.CancelFunc),
}

func (pm *PollingManager) Start(conn *models.ChannelConnection, inboundSvc *services.ChannelInboundService) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Para worker anterior se houver
	if cancel, exists := pm.workers[conn.ID]; exists {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	pm.workers[conn.ID] = cancel

	go func() {
		client := NewClient(conn.CredentialsEnc)
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

				adapter := &Adapter{}
				for _, update := range updates {
					offset = update.UpdateID + 1
					raw, _ := json.Marshal(update)
					msgs, parseErr := adapter.ParseInbound(ctx, conn, raw, nil)
					if parseErr != nil {
						log.Printf("[Telegram Polling] parse error (conn %d): %v", conn.ID, parseErr)
						continue
					}
					if inboundSvc != nil {
						for _, msg := range msgs {
							if _, rxErr := inboundSvc.Receive(ctx, msg); rxErr != nil {
								log.Printf("[Telegram Polling] receive error (conn %d): %v", conn.ID, rxErr)
							}
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
