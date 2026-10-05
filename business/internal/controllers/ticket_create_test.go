package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newCreateTicketRouter(t *testing.T, db *gorm.DB, tenantID uuid.UUID, userID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	updateUC := usecases.NewUpdateTicketUseCase(repository.NewGORMTicketRepo(db), &stubEventBus{}, nil, nil)
	tc := NewTicketController(updateUC, nil, &stubMessageRepo{}, &stubPublisher{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", "tenant")
		c.Set("userId", float64(userID))
		c.Next()
	})
	r.POST("/tickets", tc.CreateTicket)
	return r
}

func postCreateTicket(r *gin.Engine, payload map[string]interface{}) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func seedCreateTicketBase(t *testing.T, db *gorm.DB, tenantID uuid.UUID) (models.User, models.Contact) {
	t.Helper()
	user := models.User{Name: "Ana", Email: "ana-" + uuid.NewString() + "@t.io", TenantID: tenantID}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	contact := models.Contact{Name: "Cliente", Number: "5511" + uuid.NewString()[:8], TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	return user, contact
}

func seedConnection(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name, status string) models.Whatsapp {
	t.Helper()
	wa := models.Whatsapp{Name: name + "-" + uuid.NewString()[:6], TenantID: tenantID, Status: status}
	if err := db.Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	return wa
}

func decode(t *testing.T, res *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("resposta não é JSON: %v — %s", err, res.Body.String())
	}
	return out
}

func TestCreateTicket_SingleConnectedConnectionIsUsed(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, contact := seedCreateTicketBase(t, db, tenantID)
	wa := seedConnection(t, db, tenantID, "unica", "CONNECTED")
	seedConnection(t, db, tenantID, "off", "DISCONNECTED")

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantID, user.ID), map[string]interface{}{
		"contactId": contact.ID, "userId": 999, "status": "open",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d — %s", res.Code, res.Body.String())
	}
	out := decode(t, res)
	if int(out["whatsappId"].(float64)) != wa.ID {
		t.Fatalf("whatsappId = %v, esperado %d", out["whatsappId"], wa.ID)
	}
	if int(out["userId"].(float64)) != user.ID {
		t.Fatalf("userId deve vir do token (%d), não do body: %v", user.ID, out["userId"])
	}
	if out["status"] != "open" {
		t.Fatalf("status = %v", out["status"])
	}
}

func TestCreateTicket_NoConnectedConnection(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, contact := seedCreateTicketBase(t, db, tenantID)
	seedConnection(t, db, tenantID, "off", "DISCONNECTED")

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantID, user.ID), map[string]interface{}{"contactId": contact.ID})
	if res.Code != http.StatusConflict {
		t.Fatalf("esperado 409, veio %d — %s", res.Code, res.Body.String())
	}
	if code := decode(t, res)["code"]; code != "NO_CONNECTED_CONNECTION" {
		t.Fatalf("code = %v", code)
	}
}

func TestCreateTicket_MultipleConnectedAsksWhichOne(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, contact := seedCreateTicketBase(t, db, tenantID)
	a := seedConnection(t, db, tenantID, "a", "CONNECTED")
	b := seedConnection(t, db, tenantID, "b", "CONNECTED")
	r := newCreateTicketRouter(t, db, tenantID, user.ID)

	res := postCreateTicket(r, map[string]interface{}{"contactId": contact.ID})
	if res.Code != http.StatusConflict {
		t.Fatalf("esperado 409, veio %d — %s", res.Code, res.Body.String())
	}
	out := decode(t, res)
	if out["code"] != "CONNECTION_REQUIRED" {
		t.Fatalf("code = %v", out["code"])
	}
	if opts, _ := out["connections"].([]interface{}); len(opts) != 2 {
		t.Fatalf("esperado 2 opções, veio %v", out["connections"])
	}

	res = postCreateTicket(r, map[string]interface{}{"contactId": contact.ID, "whatsappId": b.ID})
	if res.Code != http.StatusCreated {
		t.Fatalf("com whatsappId escolhido esperado 201, veio %d — %s", res.Code, res.Body.String())
	}
	if int(decode(t, res)["whatsappId"].(float64)) != b.ID {
		t.Fatalf("deveria usar a conexão escolhida (%d, não %d)", b.ID, a.ID)
	}
}

func TestCreateTicket_UserPreferredConnectionWins(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	_, contact := seedCreateTicketBase(t, db, tenantID)
	seedConnection(t, db, tenantID, "a", "CONNECTED")
	b := seedConnection(t, db, tenantID, "b", "CONNECTED")
	user := models.User{Name: "Bia", Email: "bia-" + uuid.NewString() + "@t.io", TenantID: tenantID, WhatsappID: &b.ID}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantID, user.ID), map[string]interface{}{"contactId": contact.ID})
	if res.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d — %s", res.Code, res.Body.String())
	}
	if int(decode(t, res)["whatsappId"].(float64)) != b.ID {
		t.Fatalf("deveria usar a conexão preferida do usuário (%d)", b.ID)
	}
}

func TestCreateTicket_RequestedConnectionNotConnected(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, contact := seedCreateTicketBase(t, db, tenantID)
	seedConnection(t, db, tenantID, "on", "CONNECTED")
	off := seedConnection(t, db, tenantID, "off", "DISCONNECTED")

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantID, user.ID), map[string]interface{}{"contactId": contact.ID, "whatsappId": off.ID})
	if res.Code != http.StatusConflict || decode(t, res)["code"] != "CONNECTION_NOT_CONNECTED" {
		t.Fatalf("esperado 409 CONNECTION_NOT_CONNECTED, veio %d — %s", res.Code, res.Body.String())
	}
}

func TestCreateTicket_ReturnsExistingOpenTicket(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, contact := seedCreateTicketBase(t, db, tenantID)
	seedConnection(t, db, tenantID, "unica", "CONNECTED")
	r := newCreateTicketRouter(t, db, tenantID, user.ID)

	first := postCreateTicket(r, map[string]interface{}{"contactId": contact.ID})
	if first.Code != http.StatusCreated {
		t.Fatalf("1ª chamada: %d — %s", first.Code, first.Body.String())
	}
	second := postCreateTicket(r, map[string]interface{}{"contactId": contact.ID})
	if second.Code != http.StatusOK {
		t.Fatalf("2ª chamada deveria devolver o existente (200), veio %d — %s", second.Code, second.Body.String())
	}
	if decode(t, first)["id"] != decode(t, second)["id"] {
		t.Fatal("2ª chamada deveria devolver o mesmo ticket")
	}
	// Mesma forma de resposta nos dois caminhos (201 e 200): contato e usuário carregados.
	for _, res := range []*httptest.ResponseRecorder{first, second} {
		out := decode(t, res)
		if ct, _ := out["contact"].(map[string]interface{}); ct == nil || ct["name"] != "Cliente" {
			t.Fatalf("resposta deveria trazer o contato carregado: %v", out["contact"])
		}
		if us, _ := out["user"].(map[string]interface{}); us == nil || us["name"] != "Ana" {
			t.Fatalf("resposta deveria trazer o usuário carregado: %v", out["user"])
		}
	}
	var n int64
	db.Model(&models.Ticket{}).Where(`"tenantId" = ?`, tenantID).Count(&n)
	if n != 1 {
		t.Fatalf("esperado 1 ticket, existem %d", n)
	}
}

func TestCreateTicket_ContactFromOtherTenantIs404(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	userA, _ := seedCreateTicketBase(t, db, tenantA)
	_, contactB := seedCreateTicketBase(t, db, tenantB)
	seedConnection(t, db, tenantA, "unica", "CONNECTED")

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantA, userA.ID), map[string]interface{}{"contactId": contactB.ID})
	if res.Code != http.StatusNotFound {
		t.Fatalf("contato de outro tenant: esperado 404, veio %d — %s", res.Code, res.Body.String())
	}
}

func TestCreateTicket_ConnectionFromOtherTenantIsNotUsable(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	userA, contactA := seedCreateTicketBase(t, db, tenantA)
	foreign := seedConnection(t, db, tenantB, "alheia", "CONNECTED")

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantA, userA.ID), map[string]interface{}{"contactId": contactA.ID, "whatsappId": foreign.ID})
	if res.Code != http.StatusConflict {
		t.Fatalf("conexão de outro tenant: esperado 409, veio %d — %s", res.Code, res.Body.String())
	}
}

func TestCreateTicket_MissingContactIDIs400(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	user, _ := seedCreateTicketBase(t, db, tenantID)

	res := postCreateTicket(newCreateTicketRouter(t, db, tenantID, user.ID), map[string]interface{}{})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d", res.Code)
	}
}

func TestPickConnection(t *testing.T) {
	c := func(id int) models.Whatsapp { return models.Whatsapp{ID: id} }
	pref := 2

	if id, err := pickConnection([]models.Whatsapp{c(1)}, nil); err != nil || id != 1 {
		t.Fatalf("única conectada: id=%d err=%v", id, err)
	}
	if id, err := pickConnection([]models.Whatsapp{c(1), c(2)}, &pref); err != nil || id != 2 {
		t.Fatalf("preferida conectada deve vencer: id=%d err=%v", id, err)
	}
	other := 9
	if _, err := pickConnection([]models.Whatsapp{c(1), c(2)}, &other); err == nil {
		t.Fatal("preferida desconectada com 2 conectadas deve pedir escolha")
	}
	if id, err := pickConnection([]models.Whatsapp{c(1)}, &other); err != nil || id != 1 {
		t.Fatalf("preferida desconectada com 1 conectada cai na única: id=%d err=%v", id, err)
	}
}
