package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func audioToken(t *testing.T, u models.User, tenant uuid.UUID) string {
	t.Helper()
	t.Setenv("JWT_SECRET", "audio-secret")
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": u.ID, "tenantId": tenant.String(), "alcance": u.Alcance, "exp": 4102444800,
	}).SignedString([]byte("audio-secret"))
	require.NoError(t, err)
	return tok
}

type audioRig struct {
	w   *callWorld
	srv *httptest.Server
}

func newAudioRig(t *testing.T) *audioRig {
	t.Helper()
	w := newCallWorld(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctl := NewCallAudioController(w.svc, calls.NewAudio(), calls.NewEngineDialer(""), w.db)
	r.GET("/calls/:id/audio", ctl.Stream)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &audioRig{w: w, srv: srv}
}

func (a *audioRig) dial(callID, token string, hdr http.Header) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(a.srv.URL, "http") + "/calls/" + callID + "/audio"
	if token != "" {
		url += "?token=" + token
	}
	return websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: hdr})
}

func (a *audioRig) answered(t *testing.T, callID string, u models.User) {
	t.Helper()
	require.NoError(t, a.w.db.Create(&models.CallLog{
		TenantID: a.w.tenant, CallID: callID, WhatsappID: a.w.wa.ID, Direction: "incoming", Status: "active",
		HandledByUserID: &u.ID,
	}).Error)
}

func status(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func TestAudioStream_AuthMatrix(t *testing.T) {
	a := newAudioRig(t)
	owner := a.w.user(t, "dono", "proprio", a.w.tenant, "receive")
	a.answered(t, "A1", owner)
	other := a.w.user(t, "outro", "proprio", a.w.tenant, "receive")
	noPerm := a.w.user(t, "semperm", "proprio", a.w.tenant)
	intruder := a.w.user(t, "intruso", "tenant", a.w.other, "receive")

	_, resp, err := a.dial("A1", "", nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, status(resp), "sem token")

	_, resp, err = a.dial("A1", "lixo.lixo.lixo", nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, status(resp), "token inválido")

	_, resp, err = a.dial("A1", audioToken(t, noPerm, a.w.tenant), nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, status(resp), "sem calls:receive nem calls:place")

	_, resp, err = a.dial("A1", audioToken(t, other, a.w.tenant), nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, status(resp), "outro operador da mesma empresa")

	_, resp, err = a.dial("A1", audioToken(t, intruder, a.w.other), nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, status(resp), "outra empresa: 404, sem revelar que a chamada existe")

	_, resp, err = a.dial("NAO-EXISTE", audioToken(t, owner, a.w.tenant), nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, status(resp))
}

func TestAudioStream_OwnerWithoutEngineGetsClosedCleanly(t *testing.T) {
	a := newAudioRig(t)
	owner := a.w.user(t, "dono", "proprio", a.w.tenant, "receive")
	a.answered(t, "A2", owner)

	c, _, err := a.dial("A2", audioToken(t, owner, a.w.tenant), nil)
	require.NoError(t, err, "autorizado: o handshake passa")
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, err = c.Read(ctx)
	require.Error(t, err)
	assert.Equal(t, websocket.StatusInternalError, websocket.CloseStatus(err), "engine/áudio não configurado: fecha com erro claro")
}

func TestAudioStream_PlaceOnlyOperatorMayOpen(t *testing.T) {
	a := newAudioRig(t)
	caller := a.w.user(t, "ligador", "proprio", a.w.tenant, "place")
	a.answered(t, "A3", caller)
	c, _, err := a.dial("A3", audioToken(t, caller, a.w.tenant), nil)
	require.NoError(t, err, "quem liga (só calls:place) também abre o áudio")
	c.CloseNow()
}

func TestAudioOrigins_FromEnv(t *testing.T) {
	t.Setenv("CALLS_AUDIO_ORIGINS", " https://a.exemplo.com , ,https://b.exemplo.com ")
	assert.Equal(t, []string{"https://a.exemplo.com", "https://b.exemplo.com"}, audioOriginPatterns())
	t.Setenv("CALLS_AUDIO_ORIGINS", "")
	assert.Empty(t, audioOriginPatterns())
}

// Documenta o comportamento do handshake de Origin: o navegador SEMPRE manda
// Origin, e a biblioteca só aceita a mesma origem do host por padrão.
func TestAudioStream_CrossOriginIsRejectedUnlessAllowed(t *testing.T) {
	a := newAudioRig(t)
	owner := a.w.user(t, "dono", "proprio", a.w.tenant, "receive")
	a.answered(t, "A4", owner)
	tok := audioToken(t, owner, a.w.tenant)

	_, resp, err := a.dial("A4", tok, http.Header{"Origin": []string{"https://site-malicioso.example"}})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, status(resp), "origem desconhecida é recusada")

	t.Setenv("CALLS_AUDIO_ORIGINS", "https://app.exemplo.com")
	c, _, err := a.dial("A4", tok, http.Header{"Origin": []string{"https://app.exemplo.com"}})
	require.NoError(t, err, "origem permitida por configuração passa")
	c.CloseNow()
}
