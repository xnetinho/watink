package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// stubPublisher is a no-op CommandPublisher for tests.
type stubPublisher struct {
	calls []map[string]interface{}
	err   error
}

func (s *stubPublisher) PublishCommand(routingKey string, payload interface{}) error {
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, map[string]interface{}{"routingKey": routingKey, "payload": payload})
	return nil
}

func TestSendMessageTextJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)

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
	r.POST("/messages/:ticketId", mc.SendMessage)

	t.Run("happy path — text message published", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"body": "Hello", "mediaType": "", "mediaUrl": ""})
		req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — %s", res.Code, res.Body.String())
		}
		if len(pub.calls) == 0 {
			t.Fatal("expected PublishCommand to be called")
		}
	})

	t.Run("ticket not found returns 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"body": "Hello"})
		req := httptest.NewRequest(http.MethodPost, "/messages/999999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", res.Code)
		}
	})

	t.Run("invalid ticket id returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"body": "Hello"})
		req := httptest.NewRequest(http.MethodPost, "/messages/notanint", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", res.Code)
		}
	})

	t.Run("body too long returns 400", func(t *testing.T) {
		longBody := make([]byte, 65536)
		for i := range longBody {
			longBody[i] = 'x'
		}
		body, _ := json.Marshal(map[string]string{"body": string(longBody)})
		req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", res.Code)
		}
	})

	t.Run("wrong tenant cannot send message", func(t *testing.T) {
		rOther := gin.New()
		rOther.Use(testScopedMiddleware(db, uuid.New().String()))
		rOther.POST("/messages/:ticketId", mc.SendMessage)

		body, _ := json.Marshal(map[string]string{"body": "Hello"})
		req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		rOther.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-tenant, got %d", res.Code)
		}
	})
}

func TestReactToMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)

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
	msg := models.Message{
		ID:        "MSG1",
		Body:      "Olá",
		TicketID:  ticket.ID,
		FromMe:    false,
		ContactID: &contact.ID,
		TenantID:  tenantID,
		Reactions: "[]",
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/message/:messageId/react", mc.ReactToMessage)

	t.Run("happy path — publishes command and persists optimistic reaction", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"reaction": "👍"})
		req := httptest.NewRequest(http.MethodPost, "/message/MSG1/react", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — %s", res.Code, res.Body.String())
		}
		if len(pub.calls) == 0 {
			t.Fatal("expected PublishCommand to be called")
		}

		var updated models.Message
		if err := db.First(&updated, "id = ?", "MSG1").Error; err != nil {
			t.Fatal(err)
		}
		if updated.Reactions == "[]" || updated.Reactions == "" {
			t.Fatalf("expected reactions to be persisted, got %q", updated.Reactions)
		}
	})

	t.Run("empty reaction removes the bot's own reaction", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"reaction": ""})
		req := httptest.NewRequest(http.MethodPost, "/message/MSG1/react", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — %s", res.Code, res.Body.String())
		}

		var updated models.Message
		if err := db.First(&updated, "id = ?", "MSG1").Error; err != nil {
			t.Fatal(err)
		}
		if updated.Reactions != "[]" {
			t.Fatalf("expected reactions to be cleared, got %q", updated.Reactions)
		}
	})

	t.Run("message not found returns 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"reaction": "👍"})
		req := httptest.NewRequest(http.MethodPost, "/message/DOES-NOT-EXIST/react", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", res.Code)
		}
	})

	t.Run("wrong tenant cannot react", func(t *testing.T) {
		rOther := gin.New()
		rOther.Use(testScopedMiddleware(db, uuid.New().String()))
		rOther.POST("/message/:messageId/react", mc.ReactToMessage)

		body, _ := json.Marshal(map[string]string{"reaction": "👍"})
		req := httptest.NewRequest(http.MethodPost, "/message/MSG1/react", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		rOther.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-tenant, got %d", res.Code)
		}
	})
}

func TestSanitizeMimeType(t *testing.T) {
	cases := []struct {
		input    string
		wantSafe bool // just check it doesn't contain control chars
	}{
		{"image/jpeg", true},
		{"video/mp4", true},
		{"", true},                   // falls back to application/octet-stream
		{"image/jpeg\nhacked", true}, // newline stripped
	}
	for _, tc := range cases {
		got := sanitizeMimeType(tc.input)
		for _, ch := range got {
			if ch == '\n' || ch == '\r' || ch < 0x20 {
				t.Errorf("sanitizeMimeType(%q) contains control char in %q", tc.input, got)
			}
		}
		if got == "" {
			t.Errorf("sanitizeMimeType(%q) returned empty string", tc.input)
		}
	}
}

func TestMimeTypeToMediaTypeSend(t *testing.T) {
	cases := []struct{ mime, want string }{
		{"image/jpeg", "image"},
		{"video/mp4", "video"},
		{"audio/mpeg", "audio"},
		{"application/pdf", "document"},
		{"", "document"},
	}
	for _, tc := range cases {
		got := mimeTypeToMediaType(tc.mime)
		if got != tc.want {
			t.Errorf("mimeTypeToMediaType(%q) = %q, want %q", tc.mime, got, tc.want)
		}
	}
}

// Responder a uma mensagem: o frontend manda o objeto da mensagem citada em "quotedMsg".
// O business precisa (1) repassar quotedMsgId/quotedJid ao engine, que monta o ContextInfo, e
// (2) gravar QuotedMsgID na mensagem enviada. Antes nada disso acontecia e a resposta chegava
// ao contato como mensagem solta.
func TestSendMessage_ReplyCarriesQuotedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)

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
	cid := contact.ID
	orig := models.Message{ID: "ORIG-1", Body: "oi, tudo bem?", TicketID: ticket.ID, FromMe: false, ContactID: &cid, TenantID: tenantID}
	if err := db.Create(&orig).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/messages/:ticketId", mc.SendMessage)

	send := func(t *testing.T, payload map[string]interface{}) {
		t.Helper()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", res.Code, res.Body.String())
		}
	}

	t.Run("resposta leva o id da mensagem citada ao engine e grava no banco", func(t *testing.T) {
		pub.calls = nil
		send(t, map[string]interface{}{"body": "tudo ótimo", "quotedMsg": map[string]interface{}{"id": "ORIG-1", "fromMe": false}})
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		if payload["quotedMsgId"] != "ORIG-1" {
			t.Fatalf("quotedMsgId no comando = %v", payload["quotedMsgId"])
		}
		var sent models.Message
		if err := db.Where(`"fromMe" = true AND body = ?`, "tudo ótimo").First(&sent).Error; err != nil {
			t.Fatal(err)
		}
		if sent.QuotedMsgID == nil || *sent.QuotedMsgID != "ORIG-1" {
			t.Fatalf("QuotedMsgID gravado = %v", sent.QuotedMsgID)
		}
	})

	t.Run("sem citação o comando não leva quotedMsgId", func(t *testing.T) {
		pub.calls = nil
		send(t, map[string]interface{}{"body": "mensagem solta"})
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		if v, ok := payload["quotedMsgId"]; ok && v != "" {
			t.Fatalf("quotedMsgId não deveria existir: %v", v)
		}
	})

	t.Run("citar mensagem que EXISTE mas é de outro tenant é ignorado", func(t *testing.T) {
		pub.calls = nil
		other := models.Message{ID: "OUTRO-TENANT-1", Body: "x", TicketID: ticket.ID, TenantID: uuid.New()}
		if err := db.Create(&other).Error; err != nil {
			t.Fatal(err)
		}
		send(t, map[string]interface{}{"body": "cita de outro tenant", "quotedMsg": map[string]interface{}{"id": "OUTRO-TENANT-1"}})
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		if v, ok := payload["quotedMsgId"]; ok && v != "" {
			t.Fatalf("mensagem de outro tenant não pode ser citada: %v", v)
		}
	})

	t.Run("citar mensagem que EXISTE mas é de outro ticket é ignorado", func(t *testing.T) {
		pub.calls = nil
		other := models.Message{ID: "OUTRO-TICKET-1", Body: "x", TicketID: ticket.ID + 1000, TenantID: tenantID}
		if err := db.Create(&other).Error; err != nil {
			t.Fatal(err)
		}
		send(t, map[string]interface{}{"body": "cita de outro ticket", "quotedMsg": map[string]interface{}{"id": "OUTRO-TICKET-1"}})
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		if v, ok := payload["quotedMsgId"]; ok && v != "" {
			t.Fatalf("mensagem de outro ticket não pode ser citada: %v", v)
		}
	})

	t.Run("citar mensagem de outro tenant ou inexistente é ignorado, não vaza", func(t *testing.T) {
		pub.calls = nil
		send(t, map[string]interface{}{"body": "cita fantasma", "quotedMsg": map[string]interface{}{"id": "NAO-EXISTE"}})
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		if v, ok := payload["quotedMsgId"]; ok && v != "" {
			t.Fatalf("id inexistente não pode ser repassado: %v", v)
		}
	})
}

// GET /messages/:ticketId devolve a mensagem citada dentro de "quotedMsg": o frontend a mostra no balão.
func TestListMessages_ReturnsQuotedMessageObject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	mc := NewMessageController(&stubPublisher{}, nil, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	_ = db.Create(&wa).Error
	contact := models.Contact{Name: "Maria", Number: "5511999999999", TenantID: tenantID}
	_ = db.Create(&contact).Error
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	_ = db.Create(&ticket).Error
	cid := contact.ID
	_ = db.Create(&models.Message{ID: "Q-ORIG", Body: "qual o preço?", TicketID: ticket.ID, ContactID: &cid, TenantID: tenantID}).Error
	quoted := "Q-ORIG"
	_ = db.Create(&models.Message{ID: "Q-REPLY", Body: "R$ 10", TicketID: ticket.ID, FromMe: true, ContactID: &cid, QuotedMsgID: &quoted, TenantID: tenantID}).Error
	_ = db.Create(&models.Message{ID: "Q-PLAIN", Body: "sem citação", TicketID: ticket.ID, ContactID: &cid, TenantID: tenantID}).Error

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.GET("/messages/:ticketId", mc.ListMessages)
	req := httptest.NewRequest(http.MethodGet, "/messages/"+strconv.Itoa(ticket.ID), nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	var out struct {
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]interface{}{}
	for _, m := range out.Messages {
		byID[m["id"].(string)] = m
	}
	q, ok := byID["Q-REPLY"]["quotedMsg"].(map[string]interface{})
	if !ok || q["body"] != "qual o preço?" {
		t.Fatalf("a resposta deveria trazer a mensagem citada: %v", byID["Q-REPLY"]["quotedMsg"])
	}
	if byID["Q-PLAIN"]["quotedMsg"] != nil {
		t.Fatalf("mensagem sem citação não pode trazer quotedMsg: %v", byID["Q-PLAIN"]["quotedMsg"])
	}
}

// A citada só aparece se for do mesmo tenant: o id vem do banco, mas o filtro tem de ser manual.
func TestListMessages_QuotedFromAnotherTenantIsNotLeaked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	mc := NewMessageController(&stubPublisher{}, nil, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenantA, Status: "CONNECTED"}
	_ = db.Create(&wa).Error
	contact := models.Contact{Name: "Maria", Number: "5511999999999", TenantID: tenantA}
	_ = db.Create(&contact).Error
	ticket := models.Ticket{Status: "open", TenantID: tenantA, ContactID: contact.ID, WhatsappID: wa.ID}
	_ = db.Create(&ticket).Error
	cid := contact.ID
	_ = db.Create(&models.Message{ID: "SECRET-B", Body: "segredo do outro tenant", TicketID: 999, TenantID: tenantB}).Error
	quoted := "SECRET-B"
	_ = db.Create(&models.Message{ID: "REPLY-A", Body: "oi", TicketID: ticket.ID, ContactID: &cid, QuotedMsgID: &quoted, TenantID: tenantA}).Error

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantA.String()))
	r.GET("/messages/:ticketId", mc.ListMessages)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/messages/"+strconv.Itoa(ticket.ID), nil))
	if bytes.Contains(res.Body.Bytes(), []byte("segredo do outro tenant")) {
		t.Fatalf("vazou mensagem de outro tenant: %s", res.Body.String())
	}
}

type captureBroadcaster struct{ payloads []interface{} }

func (b *captureBroadcaster) EmitToRoom(_, _, event string, p interface{}) {
	if event == "appMessage" {
		b.payloads = append(b.payloads, p)
	}
}
func (b *captureBroadcaster) EmitToTenantRoom(_, _ string, _ interface{}) {}
func (b *captureBroadcaster) EmitToNamespace(_, _ string, _ interface{})  {}

// Regressão: a resposta enviada aparecia no chat SEM a citação. A mensagem nova chega ao navegador por SSE
// (appMessage) e o reducer a coloca na lista como veio; só a listagem anexava o quotedMsg. Resultado: a
// citação só aparecia depois de recarregar. O evento precisa levar a mensagem citada junto.
func TestSendMessage_Reply_EventCarriesTheQuotedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	bc := &captureBroadcaster{}
	mc := NewMessageController(pub, bc, nil)

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
	cid := contact.ID
	orig := models.Message{ID: "ORIG-9", Body: "qual o prazo?", TicketID: ticket.ID, ContactID: &cid, TenantID: tenantID}
	if err := db.Create(&orig).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/messages/:ticketId", mc.SendMessage)
	body, _ := json.Marshal(map[string]interface{}{"body": "5 dias", "quotedMsg": map[string]interface{}{"id": "ORIG-9"}})
	req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", res.Code, res.Body.String())
	}

	if len(bc.payloads) != 1 {
		t.Fatalf("esperava 1 appMessage, veio %d", len(bc.payloads))
	}
	sent := bc.payloads[0].(map[string]interface{})["message"].(models.Message)
	if sent.QuotedMsg == nil || sent.QuotedMsg.ID != "ORIG-9" || sent.QuotedMsg.Body != "qual o prazo?" {
		t.Fatalf("o evento da resposta não leva a mensagem citada: %+v", sent.QuotedMsg)
	}
}

// O contato tem LID: o chat é por LID, mas a mensagem recebida foi gravada com o número em Participant (log real).
// A citação precisa usar o endereço do chat, e levar o texto citado para o celular desenhar a caixa.
func TestSendMessage_Reply_OneToOneUsesChatAddressAndQuotedText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, nil, nil)

	wa := models.Whatsapp{Name: "WA", TenantID: tenantID, Status: "CONNECTED"}
	if err := db.Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	lid := "217407117332513@lid"
	contact := models.Contact{Name: "C", Number: "558398060007", Lid: &lid, TenantID: tenantID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	ticket := models.Ticket{Status: "open", TenantID: tenantID, ContactID: contact.ID, WhatsappID: wa.ID}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	cid := contact.ID
	orig := models.Message{ID: "ORIG-LID", Body: "Sim", TicketID: ticket.ID, ContactID: &cid, TenantID: tenantID, Participant: "558398060007@s.whatsapp.net"}
	if err := db.Create(&orig).Error; err != nil {
		t.Fatal(err)
	}
	photo := models.Message{ID: "FOTO-1", Body: "", MediaType: "image", MediaUrl: "/public/media/x.jpg", TicketID: ticket.ID, ContactID: &cid, TenantID: tenantID}
	if err := db.Create(&photo).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/messages/:ticketId", mc.SendMessage)
	post := func(quoted string) map[string]interface{} {
		pub.calls = nil
		body, _ := json.Marshal(map[string]interface{}{"body": "resp", "quotedMsg": map[string]interface{}{"id": quoted}})
		req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.Itoa(ticket.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("esperava 200, veio %d: %s", res.Code, res.Body.String())
		}
		return pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
	}

	p := post("ORIG-LID")
	if p["quotedJid"] != lid {
		t.Fatalf("quotedJid = %v, esperado o endereço do chat %s (não o número gravado em Participant)", p["quotedJid"], lid)
	}
	if p["quotedBody"] != "Sim" {
		t.Fatalf("quotedBody = %v", p["quotedBody"])
	}
	if p := post("FOTO-1"); p["quotedBody"] != "📷 Foto" {
		t.Fatalf("citar foto sem legenda deve mandar o rótulo, veio %v", p["quotedBody"])
	}
}
