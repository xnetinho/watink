package controllers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

func TestContactController_SyncContact_EngineGo_PublishesCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, tenantID := setupContactTestDB(t)
	contact := models.Contact{Name: "Ana", Number: "5511999990001", TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}

	sessions := &MockChannelSessionRepo{}
	sessions.On("FindAll", mock.Anything, tenantID).Return([]domain.ChannelSession{
		{ID: 7, Status: "CONNECTED", EngineType: "whatsmeow", TenantID: tenantID},
	}, nil)
	pub := &mockPublisher{}

	ctrl := NewContactController(&MockContactRepo{db: db}, sessions, pub, nil)

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/contacts/:contactId/sync", ctrl.SyncContact)

	req := httptest.NewRequest(http.MethodPost, "/contacts/"+strconv.Itoa(contact.ID)+"/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if !pub.called {
		t.Fatal("publisher should be called for a whatsmeow (engine-go) session")
	}
	want := "wbot." + tenantID.String() + ".7.contact.sync"
	if pub.routingKey != want {
		t.Fatalf("routing key = %q, want %q", pub.routingKey, want)
	}
}
