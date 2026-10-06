package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SSEController struct {
	hub      *services.SSEHub
	redisSvc domain.RedisService
	db       *gorm.DB
}

// NewSSEController recebe o *gorm.DB para validar, no `?rooms=`, que o usuário
// pode ver o ticket de uma sala `chat:<id>`. db pode ser nil (testes): nesse
// caso nenhuma sala de chat extra é aceita (fail-closed).
func NewSSEController(hub *services.SSEHub, redisSvc domain.RedisService, db *gorm.DB) *SSEController {
	return &SSEController{hub: hub, redisSvc: redisSvc, db: db}
}

// canSeeTicket diz se o usuário pode assinar o chat de um ticket. Alcance de
// empresa vê todos os da própria empresa; os demais seguem a mesma regra de
// visibilidade de GetScopedDB("Tickets") (atribuído a ele, fila dele, ou
// conexão ligada a uma fila dele).
func (sc *SSEController) canSeeTicket(tenantID uuid.UUID, userID int, alcance string) func(int) bool {
	return func(ticketID int) bool {
		if sc.db == nil || userID <= 0 {
			return false
		}
		q := sc.db.Session(&gorm.Session{NewDB: true}).Model(&models.Ticket{}).
			Where(`id = ? AND "tenantId" = ?`, ticketID, tenantID)
		if alcance != "tenant" && alcance != "plataforma" {
			q = q.Where(`("userId" = ? `+
				`OR "queueId" IN (SELECT queue_id FROM user_queues WHERE user_id = ?) `+
				`OR "whatsappId" IN (SELECT wq.whatsapp_id FROM whatsapp_queues wq WHERE wq.queue_id IN (SELECT queue_id FROM user_queues WHERE user_id = ?)))`,
				userID, userID, userID)
		}
		var n int64
		return q.Count(&n).Error == nil && n > 0
	}
}

// Stream godoc
// @Summary      SSE event stream
// @Description  Opens a Server-Sent Events stream for real-time updates. Auth via token query param.
// @Tags         realtime
// @Produce      text/event-stream
// @Param        token  query  string  true  "JWT token"
// @Param        rooms  query  string  false "Extra rooms (csv), filtradas por lista permitida: chat:{id} (ticket visível ao usuário), tickets:{open|pending|closed}, notification, helpdesk-kanban"
// @Router       /events [get]
func (sc *SSEController) Stream(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected alg")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	tenantID, _ := claims["tenantId"].(string)
	if tenantID == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userIDRaw := claims["id"]
	userID := 0
	switch v := userIDRaw.(type) {
	case float64:
		userID = int(v)
	case string:
		userID, _ = strconv.Atoi(v)
	}
	alcance, _ := claims["alcance"].(string)
	tenantUUID, tenantErr := uuid.Parse(tenantID)
	if tenantErr != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Salas fixas: a da própria empresa, "notification" e a sala PESSOAL do usuário
	// (derivada do token, nunca do query). Extras passam por lista permitida: sem
	// isso, `?rooms=tenant:<outra empresa>` entregava os eventos dela.
	rooms := []string{"tenant:" + tenantID, "notification"}
	if userID > 0 {
		rooms = append(rooms, userRoom(tenantUUID, userID))
	}
	rooms = append(rooms, allowedExtraRooms(c.Query("rooms"), tenantUUID, userID, sc.canSeeTicket(tenantUUID, userID, alcance))...)

	log.Printf("[SSE] connect connID rooms=%d", len(rooms)) // logs only count, not room names — no user-controlled string in format
	connID := uuid.New().String()
	ch, cleanup := sc.hub.Register(connID, rooms)
	defer cleanup()

	w := c.Writer
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Send initial connection event.
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	// Replay: if the client reconnects with Last-Event-ID, attempt to redeliver
	// the cached message payload from Redis (key wbot:msg:{lastEventID}).
	if lastEventID := c.GetHeader("Last-Event-ID"); lastEventID != "" && sc.redisSvc != nil {
		key := "wbot:msg:" + lastEventID
		if val, err := sc.redisSvc.Get(c.Request.Context(), key); err == nil && val != "" {
			_, _ = fmt.Fprintf(w, "id: %s\nevent: appMessage\ndata: %s\n\n", lastEventID, val)
			flusher.Flush()
		} else if err != nil {
			log.Printf("[SSE] replay: cache miss or error: %v", err)
		}
	}

	timeout := time.NewTimer(30 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case msg, open := <-ch:
			if !open {
				return
			}
			_, _ = fmt.Fprint(w, msg)
			flusher.Flush()
		case <-c.Request.Context().Done():
			return
		case <-timeout.C:
			return
		}
	}
}
