package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// stubEventBus is a no-op EventBus for tests.
type stubEventBus struct{}

func (s *stubEventBus) Publish(_ context.Context, _ domain.DomainEvent) error { return nil }
func (s *stubEventBus) Subscribe(_ string, _ domain.EventHandler) error       { return nil }

// stubMessageRepo is a no-op MessageRepository for tests.
type stubMessageRepo struct{}

func (s *stubMessageRepo) Create(_ context.Context, _ *domain.Message) error { return nil }
func (s *stubMessageRepo) CreateIfNotExists(_ context.Context, _ *domain.Message) error {
	return nil
}
func (s *stubMessageRepo) FindByID(_ context.Context, _ string, _ uuid.UUID) (*domain.Message, error) {
	return nil, nil
}
func (s *stubMessageRepo) FindOldestByTicket(_ context.Context, _ int, _ uuid.UUID) (*domain.Message, error) {
	return nil, nil
}
func (s *stubMessageRepo) ExistsByID(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (s *stubMessageRepo) Update(_ context.Context, _ *domain.Message, _ map[string]interface{}) error {
	return nil
}

func TestUpdateTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	ticketRepo := repository.NewGORMTicketRepo(db)
	updateUC := usecases.NewUpdateTicketUseCase(ticketRepo, &stubEventBus{}, nil, nil)
	tc := NewTicketController(updateUC, nil, &stubMessageRepo{}, &stubPublisher{})

	// Seed whatsapp + contact + ticket
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	if err := db.Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	ticket := models.Ticket{
		Status:     "pending",
		TenantID:   tenantID,
		ContactID:  contact.ID,
		WhatsappID: wa.ID,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", "tenant")
		c.Set("userId", float64(1)) // float64 as Gin parses JSON numbers
		c.Next()
	})
	r.PUT("/tickets/:ticketId", tc.UpdateTicket)

	t.Run("happy path — updates status", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"status": "open"})
		req := httptest.NewRequest(http.MethodPut, "/tickets/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — %s", res.Code, res.Body.String())
		}
	})

	t.Run("ticket not found returns 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"status": "open"})
		req := httptest.NewRequest(http.MethodPut, "/tickets/999999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", res.Code)
		}
	})

	// M18/GAP-VAL-3: status sem ValidateStringField aceitava qualquer string
	// livre, sem limite de tamanho — prova o invariante agora aplicado.
	t.Run("status too long returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"status": strings.Repeat("x", 51)})
		req := httptest.NewRequest(http.MethodPut, "/tickets/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for oversized status, got %d — %s", res.Code, res.Body.String())
		}
	})
}

// O frontend zera o contador de não lidas com PUT /tickets/:id {"unreadMessages": 0}. O handler só lia
// status/userId/queueId e descartava o campo: o contador nunca zerava, mesmo lendo a conversa.
func TestUpdateTicket_ResetsUnreadMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID, otherTenant := uuid.New(), uuid.New()
	ticketRepo := repository.NewGORMTicketRepo(db)
	updateUC := usecases.NewUpdateTicketUseCase(ticketRepo, &stubEventBus{}, nil, nil)
	tc := NewTicketController(updateUC, nil, &stubMessageRepo{}, &stubPublisher{})

	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	_ = db.Create(&wa).Error
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	_ = db.Create(&contact).Error
	mine := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID, UnreadMessages: 5}
	_ = db.Create(&mine).Error
	other := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID, UnreadMessages: 7}
	_ = db.Create(&other).Error

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", "tenant")
		c.Set("userId", float64(1))
		c.Next()
	})
	r.PUT("/tickets/:ticketId", tc.UpdateTicket)
	put := func(id int, payload map[string]interface{}) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/tickets/"+strconv.Itoa(id), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	unread := func(id int) int {
		var n int
		db.Raw(`SELECT "unreadMessages" FROM "Tickets" WHERE id = ?`, id).Scan(&n)
		return n
	}

	if res := put(mine.ID, map[string]interface{}{"unreadMessages": 0}); res.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", res.Code, res.Body.String())
	}
	if got := unread(mine.ID); got != 0 {
		t.Fatalf("o contador devia zerar, ficou %d", got)
	}
	if got := unread(other.ID); got != 7 {
		t.Fatalf("zerar um ticket não pode mexer em outro: %d", got)
	}

	t.Run("só aceita zerar: um valor positivo vindo do cliente é ignorado", func(t *testing.T) {
		_ = db.Model(&models.Ticket{}).Where("id = ?", mine.ID).Update("unreadMessages", 3).Error
		put(mine.ID, map[string]interface{}{"unreadMessages": 99})
		if got := unread(mine.ID); got != 3 {
			t.Fatalf("o cliente não define o contador: %d", got)
		}
	})

	t.Run("zerar não atinge ticket de outro tenant", func(t *testing.T) {
		foreign := models.Ticket{Status: "open", TenantID: otherTenant, ContactID: contact.ID, WhatsappID: wa.ID, UnreadMessages: 4}
		_ = db.Create(&foreign).Error
		if res := put(foreign.ID, map[string]interface{}{"unreadMessages": 0}); res.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, veio %d", res.Code)
		}
		if got := unread(foreign.ID); got != 4 {
			t.Fatalf("ticket de outro tenant foi alterado: %d", got)
		}
	})
}

// DeleteTicket had no backend route at all (issue #413) — the frontend
// already called DELETE /tickets/:ticketId, which 404'd unconditionally.
func TestDeleteTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	ticketRepo := repository.NewGORMTicketRepo(db)
	updateUC := usecases.NewUpdateTicketUseCase(ticketRepo, &stubEventBus{}, nil, nil)
	tc := NewTicketController(updateUC, nil, &stubMessageRepo{}, &stubPublisher{})

	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	if err := db.Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	msg := models.Message{ID: "msg-del-1", Body: "oi", TicketID: ticket.ID, TenantID: tenantID}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := models.Pipeline{Name: "P1", TenantID: tenantID}
	if err := db.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	stage := models.PipelineStage{Name: "S1", PipelineID: pipeline.ID}
	if err := db.Create(&stage).Error; err != nil {
		t.Fatal(err)
	}
	deal := models.Deal{Name: "D1", StageID: stage.ID, ContactID: contact.ID, TicketID: &ticket.ID, TenantID: tenantID}
	if err := db.Create(&deal).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.DELETE("/tickets/:ticketId", tc.DeleteTicket)

	t.Run("happy path — deletes ticket and messages, detaches deal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tickets/"+strconv.Itoa(ticket.ID), nil)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — %s", res.Code, res.Body.String())
		}

		var ticketCount, msgCount int64
		db.Model(&models.Ticket{}).Where("id = ?", ticket.ID).Count(&ticketCount)
		db.Model(&models.Message{}).Where("id = ?", msg.ID).Count(&msgCount)
		if ticketCount != 0 {
			t.Error("expected ticket to be deleted")
		}
		if msgCount != 0 {
			t.Error("expected ticket's messages to be deleted")
		}

		var reloadedDeal models.Deal
		if err := db.First(&reloadedDeal, deal.ID).Error; err != nil {
			t.Fatalf("deal should survive ticket deletion: %v", err)
		}
		if reloadedDeal.TicketID != nil {
			t.Error("expected deal.ticketId to be detached (nil), not the deal deleted")
		}
	})

	t.Run("ticket not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tickets/999999", nil)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", res.Code)
		}
	})
}

func TestRecoverHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	pub := &stubPublisher{}
	ticketRepo := repository.NewGORMTicketRepo(db)
	updateUC := usecases.NewUpdateTicketUseCase(ticketRepo, &stubEventBus{}, nil, nil)
	tc := NewTicketController(updateUC, nil, &stubMessageRepo{}, pub)

	// Seed whatsapp + contact with number + ticket
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	if err := db.Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	contact := models.Contact{Name: "C", Number: "5511999999999", TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	ticket := models.Ticket{
		Status:     "open",
		TenantID:   tenantID,
		ContactID:  contact.ID,
		WhatsappID: wa.ID,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/tickets/:ticketId/history/recover", tc.RecoverHistory)

	t.Run("happy path — publishes recover command", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"range": "7d"})
		req := httptest.NewRequest(http.MethodPost, "/tickets/"+strconv.Itoa(ticket.ID)+"/history/recover", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d — %s", res.Code, res.Body.String())
		}
		if len(pub.calls) == 0 {
			t.Fatal("expected PublishCommand to be called")
		}
	})

	t.Run("ticket not found returns 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"range": "1d"})
		req := httptest.NewRequest(http.MethodPost, "/tickets/999999/history/recover", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", res.Code)
		}
	})

	// M18/GAP-VAL-3: range sem ValidateStringField aceitava qualquer string
	// livre, sem limite de tamanho — prova o invariante agora aplicado.
	t.Run("range too long returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"range": strings.Repeat("x", 21)})
		req := httptest.NewRequest(http.MethodPost, "/tickets/"+strconv.Itoa(ticket.ID)+"/history/recover", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for oversized range, got %d — %s", res.Code, res.Body.String())
		}
	})

	t.Run("invalid ticket id returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"range": "1d"})
		req := httptest.NewRequest(http.MethodPost, "/tickets/notanint/history/recover", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", res.Code)
		}
	})
}
