package controllers

import (
	"errors"
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/enginego"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/gin-gonic/gin"
)

// WAVersionController expõe em Configurações (superadmin) a versão do cliente WhatsApp que o engine anuncia: as versões
// conhecidas lado a lado e a escolha entre automático e manual.
type WAVersionController struct {
	svc *services.WAVersionService
}

func NewWAVersionController(svc *services.WAVersionService) *WAVersionController {
	return &WAVersionController{svc: svc}
}

// Get devolve a escolha salva e, quando o engine responde, as versões que ele enxerga. Se o engine não responde, a escolha
// ainda vem (engineError preenchido) para a tela não ficar vazia.
//
// @Summary      Versão do cliente WhatsApp
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /system/wa-version [get]
func (ctrl *WAVersionController) Get(c *gin.Context) {
	set, err := ctrl.svc.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wa_version_read_failed"})
		return
	}
	resp := gin.H{"mode": set.Mode, "manualVersion": set.ManualVersion, "updatedAt": set.UpdatedAt}
	if st, err := enginego.GetWAVersionStatus(c.Request.Context()); err != nil {
		resp["engineError"] = "engine_unavailable"
	} else {
		resp["engine"] = st
	}
	c.JSON(http.StatusOK, resp)
}

type waVersionBody struct {
	Mode          string `json:"mode"`
	ManualVersion string `json:"manualVersion"`
}

// Save grava a escolha e manda o engine aplicá-la. A gravação vale mesmo que o engine esteja fora: ele lê a tabela ao
// subir (engineApplied=false avisa a tela).
//
// @Summary      Define a versão do cliente WhatsApp
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /system/wa-version [put]
func (ctrl *WAVersionController) Save(c *gin.Context) {
	var body waVersionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	set, err := ctrl.svc.Save(body.Mode, body.ManualVersion)
	switch {
	case errors.Is(err, services.ErrInvalidWAVersion):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_wa_version"})
		return
	case errors.Is(err, services.ErrInvalidWAVersionMode):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_wa_version_mode"})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wa_version_save_failed"})
		return
	}
	resp := gin.H{"mode": set.Mode, "manualVersion": set.ManualVersion, "updatedAt": set.UpdatedAt}
	if st, err := enginego.RefreshWAVersion(c.Request.Context()); err != nil {
		resp["engineApplied"] = false
	} else {
		resp["engineApplied"] = true
		resp["engine"] = st
	}
	c.JSON(http.StatusOK, resp)
}
