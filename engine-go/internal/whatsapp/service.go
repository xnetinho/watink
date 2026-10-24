package whatsapp

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/alltomatos/watinkdev/engine-go/internal/calls"
	"github.com/alltomatos/watinkdev/engine-go/internal/rabbitmq"
	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// WhatsAppService manages all active WhatsApp client sessions for a single engine-go instance.
type clientKey struct {
	tenant string
	id     int
}

type WhatsAppService struct {
	container       *sqlstore.Container
	clients         map[int]*whatsmeow.Client
	clientsByTenant map[clientKey]*whatsmeow.Client
	mu              sync.RWMutex
	rabbit          *rabbitmq.RabbitMQService

	historyMu       sync.Mutex
	historyRequests map[string]*pendingHistory

	groupNameMu  sync.Mutex
	groupNames   map[string]string
	groupMetaMu  sync.Mutex
	groupMetaMap map[string]groupMeta

	picMu    sync.Mutex
	picCache map[string]string // JID string → profile picture URL

	callMu       sync.Mutex
	callSessions map[int]*calls.Session

	// publishEvent routes an event envelope to the messaging broker.
	// Settable in tests to capture emitted events without a real broker.
	publishEvent func(tenantID string, sessionID int, eventType string, payload map[string]interface{})
}

// groupMeta holds cached community/sub-group metadata for a group JID.
type groupMeta struct {
	isCommunity bool
	isSubGroup  bool
}

// NewWhatsAppService creates a WhatsAppService connecting whatsmeow to PostgreSQL via sqlstore.
func NewWhatsAppService(rabbit *rabbitmq.RabbitMQService) *WhatsAppService {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	dsn := BuildPostgresDSN()

	container, err := sqlstore.New(context.Background(), "postgres", dsn, dbLog)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres for WhatsMeow store: %v", err)
	}

	svc := &WhatsAppService{
		container:       container,
		clients:         make(map[int]*whatsmeow.Client),
		clientsByTenant: make(map[clientKey]*whatsmeow.Client),
		rabbit:          rabbit,
		historyRequests: make(map[string]*pendingHistory),
		groupNames:      make(map[string]string),
		groupMetaMap:    make(map[string]groupMeta),
		picCache:        make(map[string]string),
		callSessions:    make(map[int]*calls.Session),
	}
	svc.publishEvent = svc.defaultPublishEvent
	return svc
}

// getConnectedClient returns the client for sessionID if it is connected and logged in.
func (s *WhatsAppService) getConnectedClient(sessionID int, tenantIDs ...string) (*whatsmeow.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var client *whatsmeow.Client
	if len(tenantIDs) > 0 && tenantIDs[0] != "" {
		client = s.clientsByTenant[clientKey{tenantIDs[0], sessionID}]
	} else {
		client = s.clients[sessionID]
		if client == nil {
			for key, candidate := range s.clientsByTenant {
				if key.id == sessionID {
					if client != nil {
						return nil, fmt.Errorf("%w: session %d is ambiguous across tenants", ErrSessionNotConnected, sessionID)
					}
					client = candidate
				}
			}
		}
	}
	if client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return nil, fmt.Errorf("%w: session %d", ErrSessionNotConnected, sessionID)
	}
	return client, nil
}

// emitStatus publishes a session.status event to the backend.
func (s *WhatsAppService) emitStatus(id int, tenantID, status string) {
	s.publishEvent(tenantID, id, "session.status", map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", id),
		"status":    status,
	})
}

// emitAck publishes a message.ack event (ack: 1=sent, 2=delivered, 3=read, 4=played, 5=error).
func (s *WhatsAppService) emitAck(sessionID int, tenantID, messageID string, ack int) {
	if messageID == "" {
		return
	}
	s.publishEvent(tenantID, sessionID, "message.ack", map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", sessionID),
		"messageId": messageID,
		"ack":       ack,
	})
}

// defaultPublishEvent wraps payload in an envelope and routes it to wbot.events via RabbitMQ.
func (s *WhatsAppService) defaultPublishEvent(tenantID string, sessionID int, eventType string, payload map[string]interface{}) {
	envelope := map[string]interface{}{
		"id":        fmt.Sprintf("%d-%d", time.Now().UnixNano(), sessionID),
		"timestamp": time.Now().UnixMilli(),
		"tenantId":  tenantID,
		"type":      eventType,
		"payload":   eventPayloadWithTenant(tenantID, payload),
	}
	if err := s.rabbit.PublishEvent(wbot.Event(tenantID, strconv.Itoa(sessionID), eventType), envelope); err != nil {
		log.Printf("Failed to publish %s for session %d: %v", eventType, sessionID, err)
	}
}

// BuildPostgresDSN monta a conexão do engine. O papel do engine (watink_engine) é dono do schema "whatsmeow" e só tem
// SELECT em public."Whatsapps"; o search_path o aponta para o schema próprio, onde a biblioteca cria e usa as suas
// tabelas (ela não qualifica nomes). DB_SEARCH_PATH vazio mantém o comportamento antigo (public), para testes.
func BuildPostgresDSN() string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	if sp := os.Getenv("DB_SEARCH_PATH"); sp != "" {
		dsn += "&search_path=" + sp
	}
	return dsn
}

func eventPayloadWithTenant(tenantID string, payload map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(payload)+1)
	for k, v := range payload {
		out[k] = v
	}
	out["tenantId"] = tenantID
	return out
}
