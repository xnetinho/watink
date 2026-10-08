package controllers

import (
	"errors"
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
)

func respondRecordingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, calls.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "gravação não encontrada"})
	case errors.Is(err, calls.ErrRecordingNoS3):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "esta instalação não tem armazenamento para gravações", "code": "NO_STORAGE"})
	case errors.Is(err, calls.ErrRecordingOff):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "a gravação de chamadas está desligada", "code": "RECORDING_OFF"})
	case errors.Is(err, calls.ErrRecordingNotOpt):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "a gravação por chamada só existe no modo opcional", "code": "NOT_OPTIONAL"})
	case errors.Is(err, calls.ErrRecordingActive), errors.Is(err, calls.ErrRecordingNone), errors.Is(err, calls.ErrCallNotInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "RECORDING_STATE"})
	case errors.Is(err, calls.ErrAckRequired):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "code": "ACK_REQUIRED"})
	case errors.Is(err, calls.ErrBadMode):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_MODE"})
	default:
		utils.RespondWithInternalError(c, err, "CallController.recording")
	}
}

// StartRecording godoc
// @Summary      Iniciar gravação da chamada (modo opcional)
// @Tags         calls
// @Param        id path string true "ID da chamada"
// @Success      204
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      422 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/recording/start [post]
func (cc *CallController) StartRecording(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	if err := cc.svc.StartRecording(tenantID, userID, c.Param("id"), false); err != nil {
		respondRecordingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// StopRecording godoc
// @Summary      Parar a gravação da chamada (modo opcional)
// @Tags         calls
// @Param        id path string true "ID da chamada"
// @Success      204
// @Security     BearerAuth
// @Router       /calls/{id}/recording/stop [post]
func (cc *CallController) StopRecording(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	if err := cc.svc.StopRecording(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondRecordingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListenRecording godoc
// @Summary      Ouvir a gravação
// @Description  Devolve uma URL assinada e temporária e registra quem ouviu. Fora do alcance do usuário responde 404.
// @Tags         calls
// @Produce      json
// @Param        id path string true "ID da chamada"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/recording [get]
func (cc *CallController) ListenRecording(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	url, err := cc.svc.ListenURL(c.Request.Context(), historyScope(c, tenantID, userID), c.Param("id"))
	if err != nil {
		respondRecordingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url, "expiresInSeconds": int(calls.RecordingURLTTL.Seconds())})
}

// DeleteRecording godoc
// @Summary      Excluir a gravação
// @Tags         calls
// @Param        id path string true "ID da chamada"
// @Success      204
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/recording [delete]
func (cc *CallController) DeleteRecording(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	if err := cc.svc.DeleteRecording(c.Request.Context(), historyScope(c, tenantID, userID), c.Param("id")); err != nil {
		respondRecordingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type recordingConfigRequest struct {
	Mode string `json:"mode" binding:"required"`
	// Ack confirma que o administrador leu e aceitou o termo de responsabilidade.
	Ack bool `json:"ack"`
}

// GetRecordingConfig godoc
// @Summary      Configuração de gravação de chamadas
// @Tags         calls
// @Produce      json
// @Success      200 {object} calls.RecordingConfig
// @Security     BearerAuth
// @Router       /calls/recording-config [get]
func (cc *CallController) GetRecordingConfig(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	c.JSON(http.StatusOK, cc.svc.RecordingConfig(tenantID))
}

// PutRecordingConfig godoc
// @Summary      Alterar o modo de gravação de chamadas
// @Description  Sair de "off" exige ack=true (aceite do termo); o aceite grava usuário e horário.
// @Tags         calls
// @Accept       json
// @Produce      json
// @Param        body body recordingConfigRequest true "Modo e aceite"
// @Success      200 {object} calls.RecordingConfig
// @Failure      422 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/recording-config [put]
func (cc *CallController) PutRecordingConfig(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	var req recordingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	if err := cc.svc.SetRecordingMode(tenantID, userID, req.Mode, req.Ack); err != nil {
		respondRecordingError(c, err)
		return
	}
	c.JSON(http.StatusOK, cc.svc.RecordingConfig(tenantID))
}
