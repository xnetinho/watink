package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CallAudioController serve o WebSocket de áudio do navegador. Fica FORA do grupo
// com IsAuth porque o navegador não consegue mandar o header Authorization num
// WebSocket: a autenticação é o JWT na query (mesmo desenho do SSE), validada aqui
// com os mesmos critérios, e depois conferida a permissão e a posse da chamada.
type CallAudioController struct {
	svc   *calls.Service
	audio *calls.Audio
	dial  calls.EngineDialer
	db    *gorm.DB
}

func NewCallAudioController(svc *calls.Service, audio *calls.Audio, dial calls.EngineDialer, db *gorm.DB) *CallAudioController {
	return &CallAudioController{svc: svc, audio: audio, dial: dial, db: db}
}

type audioIdentity struct {
	userID   int
	tenantID uuid.UUID
	alcance  string
}

func parseAudioToken(raw string) (*audioIdentity, error) {
	if raw == "" {
		return nil, errors.New("sem token")
	}
	secret := os.Getenv("JWT_SECRET")
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected alg")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid || secret == "" {
		return nil, errors.New("token inválido")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("claims inválidos")
	}
	tenant, err := uuid.Parse(fmt.Sprint(claims["tenantId"]))
	if err != nil {
		return nil, errors.New("tenant inválido")
	}
	id := 0
	switch v := claims["id"].(type) {
	case float64:
		id = int(v)
	case string:
		id, _ = strconv.Atoi(v)
	}
	if id <= 0 {
		return nil, errors.New("usuário inválido")
	}
	alcance, _ := claims["alcance"].(string)
	return &audioIdentity{userID: id, tenantID: tenant, alcance: alcance}, nil
}

// Stream godoc
// @Summary      Áudio da chamada (WebSocket)
// @Description  Abre o canal de áudio PCM 16 kHz mono Int16 (quadros de 640 bytes) do operador que assumiu a chamada. Auth por token na query, como o SSE.
// @Tags         calls
// @Param        id    path  string true "ID da chamada"
// @Param        token query string true "JWT"
// @Success      101
// @Failure      401
// @Failure      403
// @Failure      404
// @Router       /calls/{id}/audio [get]
func (ca *CallAudioController) Stream(c *gin.Context) {
	id, err := parseAudioToken(c.Query("token"))
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if !auth.UserHasPermission(ca.db, id.userID, id.tenantID, "calls", "receive") &&
		!auth.UserHasPermission(ca.db, id.userID, id.tenantID, "calls", "place") {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	callID := c.Param("id")
	switch err := ca.svc.AuthorizeAudio(id.tenantID, id.userID, callID); {
	case errors.Is(err, calls.ErrNotFound):
		c.AbortWithStatus(http.StatusNotFound)
		return
	case err != nil:
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: audioOriginPatterns()})
	if err != nil {
		return
	}
	ca.svc.ServeAudio(c.Request.Context(), ca.audio, ca.dial, conn, id.tenantID, id.userID, callID)
}

// audioOriginPatterns limita as origens aceitas no handshake. Sem configuração a
// biblioteca só aceita a mesma origem do host (frontend servido junto da API);
// CALLS_AUDIO_ORIGINS (csv) acrescenta outras.
func audioOriginPatterns() []string {
	var out []string
	for _, o := range strings.Split(os.Getenv("CALLS_AUDIO_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}
