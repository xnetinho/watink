package controllers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gravação x arquivo de áudio: o comando que vai ao engine muda, a mensagem gravada continua "audio".
func TestSendMessage_AudioKind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	post := func(t *testing.T, kind string) (models.Message, map[string]interface{}) {
		db := testutil.NewTestDB(t)
		tenant := uuid.New()
		pub := &stubPublisher{}
		mc := NewMessageController(pub, &captureBroadcaster{}, nil)
		wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
		require.NoError(t, db.Create(&wa).Error)
		ct := models.Contact{Name: "C", Number: "5511", TenantID: tenant, ConnectionSource: "whatsapp", ConnectionID: wa.ID}
		require.NoError(t, db.Create(&ct).Error)
		tk := models.Ticket{TenantID: tenant, ContactID: ct.ID, WhatsappID: wa.ID, Status: "open"}
		require.NoError(t, db.Create(&tk).Error)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("db", db)
			c.Set("tenantId", tenant.String())
			c.Set("userId", float64(1))
			c.Set("alcance", "tenant")
			c.Next()
		})
		r.POST("/messages/:ticketId", mc.SendMessage)

		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		if kind != "" {
			_ = mw.WriteField("audioKind", kind)
		}
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Disposition", `form-data; name="medias"; filename="a.mp3"`)
		hdr.Set("Content-Type", "audio/mpeg")
		fw, _ := mw.CreatePart(hdr)
		_, _ = fw.Write([]byte("ID3"))
		_ = mw.Close()
		req, _ := http.NewRequest("POST", "/messages/"+itoa(tk.ID), body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var saved models.Message
		require.NoError(t, db.Where(`"fromMe" = true AND "ticketId" = ?`, tk.ID).First(&saved).Error)
		require.Len(t, pub.calls, 1)
		cmd := pub.calls[0]["payload"].(map[string]interface{})
		return saved, cmd["payload"].(map[string]interface{})
	}

	for kind, want := range map[string]string{"voice": "voice", "file": "audio", "": "audio"} {
		saved, payload := post(t, kind)
		assert.Equal(t, want, payload["mediaType"], "audioKind=%q: comando ao engine", kind)
		assert.Equal(t, "audio", saved.MediaType, "audioKind=%q: a mensagem gravada continua audio (o chat desenha igual)", kind)
	}
}

// Encaminhar uma gravação continua sendo gravação (e um arquivo de áudio continua sendo arquivo).
func TestForward_AudioMantemSeEhGravacaoOuArquivo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, &captureBroadcaster{}, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	mk := func(num string) (models.Contact, models.Ticket) {
		c := models.Contact{Name: num, Number: num, TenantID: tenant, ConnectionSource: "whatsapp", ConnectionID: wa.ID}
		require.NoError(t, db.Create(&c).Error)
		tk := models.Ticket{TenantID: tenant, ContactID: c.ID, WhatsappID: wa.ID, Status: "open"}
		require.NoError(t, db.Create(&tk).Error)
		return c, tk
	}
	_, srcTk := mk("5511111")
	dest, _ := mk("5522222")
	for id, dj := range map[string]string{"voz": `{"mimetype":"audio/mpeg","voice":true}`, "arquivo": `{"mimetype":"audio/mpeg"}`} {
		require.NoError(t, db.Create(&models.Message{ID: id, TicketID: srcTk.ID, TenantID: tenant, MediaType: "audio", MediaUrl: "/public/media/a.mp3", DataJson: dj, Body: ""}).Error)
	}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant.String())
		c.Set("userId", float64(1))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.POST("/message/:messageId/forward", mc.ForwardMessage)

	for id, want := range map[string]string{"voz": "voice", "arquivo": "audio"} {
		pub.calls = nil
		req, _ := http.NewRequest("POST", "/message/"+id+"/forward", bytes.NewBufferString(`{"contactIds":[`+itoa(dest.ID)+`],"mode":"copy"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Len(t, pub.calls, 1, id)
		payload := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		assert.Equal(t, want, payload["mediaType"], "origem=%s", id)
	}
}

func TestSanitizeFileName(t *testing.T) {
	long := strings.Repeat("a", 200) + ".pdf"
	for in, want := range map[string]string{
		"BOL-SEC-GEN-0001_ZDI-CAN-33367.pdf": "BOL-SEC-GEN-0001_ZDI-CAN-33367.pdf",
		"relatório março.pdf":                "relatório março.pdf",
		"../../etc/passwd":                   "....etcpasswd",
		`C:\\Users\\x\\a.pdf`:                "C:Usersxa.pdf",
		"a\r\nb.pdf":                         "ab.pdf",
		"..":                                 "",
		"":                                   "",
	} {
		assert.Equal(t, want, sanitizeFileName(in), "entrada %q", in)
	}
	got := sanitizeFileName(long)
	assert.Len(t, []rune(got), 120)
	assert.True(t, strings.HasSuffix(got, ".pdf"), "o fim (extensão) é preservado")
}

// O destinatário vê o NOME do arquivo que o operador escolheu, e o encaminhamento o repassa.
func TestSendMessage_DocumentoEnviaONomeOriginalEEncaminhaComOMesmoNome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, &captureBroadcaster{}, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	mk := func(num string) (models.Contact, models.Ticket) {
		c := models.Contact{Name: num, Number: num, TenantID: tenant, ConnectionSource: "whatsapp", ConnectionID: wa.ID}
		require.NoError(t, db.Create(&c).Error)
		tk := models.Ticket{TenantID: tenant, ContactID: c.ID, WhatsappID: wa.ID, Status: "open"}
		require.NoError(t, db.Create(&tk).Error)
		return c, tk
	}
	_, srcTk := mk("5511111")
	dest, _ := mk("5522222")
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant.String())
		c.Set("userId", float64(1))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.POST("/messages/:ticketId", mc.SendMessage)
	r.POST("/message/:messageId/forward", mc.ForwardMessage)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="medias"; filename="Contrato Final.pdf"`)
	hdr.Set("Content-Type", "application/pdf")
	fw, _ := mw.CreatePart(hdr)
	_, _ = fw.Write([]byte("%PDF-1.4"))
	_ = mw.Close()
	req, _ := http.NewRequest("POST", "/messages/"+itoa(srcTk.ID), body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	payloadOf := func(i int) map[string]interface{} {
		return pub.calls[i]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
	}
	require.Len(t, pub.calls, 1)
	assert.Equal(t, "Contrato Final.pdf", payloadOf(0)["fileName"], "o destinatário vê o nome do arquivo, não o hash")

	var sent models.Message
	require.NoError(t, db.Where(`"fromMe" = true AND "ticketId" = ?`, srcTk.ID).First(&sent).Error)
	req, _ = http.NewRequest("POST", "/message/"+sent.ID+"/forward", bytes.NewBufferString(`{"contactIds":[`+itoa(dest.ID)+`],"mode":"copy"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, pub.calls, 2)
	assert.Equal(t, "Contrato Final.pdf", payloadOf(1)["fileName"], "encaminhar mantém o nome")
}

// O operador escolhe como o áudio encaminhado sai (voz x arquivo), nos DOIS modos (encaminhada ou nova); sem escolha,
// mantém o tipo da origem.
func TestForward_AudioKindEscolhidoPeloOperador(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &stubPublisher{}
	mc := NewMessageController(pub, &captureBroadcaster{}, nil)
	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	mk := func(num string) (models.Contact, models.Ticket) {
		c := models.Contact{Name: num, Number: num, TenantID: tenant, ConnectionSource: "whatsapp", ConnectionID: wa.ID}
		require.NoError(t, db.Create(&c).Error)
		tk := models.Ticket{TenantID: tenant, ContactID: c.ID, WhatsappID: wa.ID, Status: "open"}
		require.NoError(t, db.Create(&tk).Error)
		return c, tk
	}
	_, srcTk := mk("5511111")
	dest, _ := mk("5522222")
	for id, dj := range map[string]string{"voz": `{"mimetype":"audio/ogg","voice":true}`, "arquivo": `{"mimetype":"audio/mpeg"}`} {
		require.NoError(t, db.Create(&models.Message{ID: id, TicketID: srcTk.ID, TenantID: tenant, MediaType: "audio", MediaUrl: "/public/media/a.mp3", DataJson: dj}).Error)
	}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant.String())
		c.Set("userId", float64(1))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.POST("/message/:messageId/forward", mc.ForwardMessage)

	send := func(src, mode, kind string) (int, string, bool) {
		pub.calls = nil
		body := `{"contactIds":[` + itoa(dest.ID) + `],"mode":"` + mode + `"`
		if kind != "" {
			body += `,"audioKind":"` + kind + `"`
		}
		req, _ := http.NewRequest("POST", "/message/"+src+"/forward", bytes.NewBufferString(body+`}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if len(pub.calls) != 1 {
			return w.Code, "", false
		}
		p := pub.calls[0]["payload"].(map[string]interface{})["payload"].(map[string]interface{})
		forwarded, _ := p["forwarded"].(bool)
		return w.Code, p["mediaType"].(string), forwarded
	}

	for _, mode := range []string{"copy", "forward"} {
		for _, c := range []struct{ src, kind, want string }{
			{"voz", "", "voice"},          // sem escolha: mantém a origem
			{"arquivo", "", "audio"},      // idem
			{"voz", "file", "audio"},      // voz virando arquivo
			{"arquivo", "voice", "voice"}, // arquivo virando voz
			{"voz", "voice", "voice"},
			{"arquivo", "file", "audio"},
		} {
			code, mt, fw := send(c.src, mode, c.kind)
			assert.Equal(t, http.StatusOK, code, "%s/%s/%s", mode, c.src, c.kind)
			assert.Equal(t, c.want, mt, "modo=%s origem=%s escolha=%q", mode, c.src, c.kind)
			assert.Equal(t, mode == "forward", fw, "o selo de encaminhada só vai no modo forward (modo=%s)", mode)
		}
	}
	code, _, _ := send("voz", "copy", "banana")
	assert.Equal(t, http.StatusBadRequest, code, "audioKind inválido é recusado")
}
