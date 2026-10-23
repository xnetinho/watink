package controllers

import (
	"net/http"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/plugins"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PluginController gerencia a ALOCAÇÃO de plugins por empresa. Não há marketplace nem licença: o administrador da
// empresa liga ou desliga cada plugin embarcado, e a alocação (PluginInstallations.active) é o único gate de runtime.
type PluginController struct {
	planLimitSvc domain.PlanLimitServiceInterface
	db           *gorm.DB
	registry     *plugins.PluginRegistry
}

// NewPluginController é montado por injeção de dependência.
func NewPluginController(planLimitSvc domain.PlanLimitServiceInterface, db *gorm.DB, registry *plugins.PluginRegistry) *PluginController {
	return &PluginController{planLimitSvc: planLimitSvc, db: db, registry: registry}
}

// @Summary      Plugins ativos da empresa
// @Tags         plugins
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /plugins/installed [get]
func (pc *PluginController) Installed(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Plugins")
	if !ok {
		return
	}

	var installs []models.PluginInstallation
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where(`"tenantId" = ? AND active = ?`, tenantID, true).
		Find(&installs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load plugin installations"})
		return
	}

	active := make([]string, 0, len(installs))
	statuses := make(map[string]string, len(installs))
	for _, inst := range installs {
		active = append(active, inst.PluginID)
		statuses[inst.PluginID] = string(pc.registry.GetStatus(tenantID, inst.PluginID))
	}

	c.JSON(http.StatusOK, gin.H{"active": active, "statuses": statuses})
}

// @Summary      Ativar plugin
// @Description  Ativa/aloca um plugin embarcado para a empresa atual.
// @Tags         plugins
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /plugins/{slug}/activate [post]
func (pc *PluginController) Activate(c *gin.Context) {
	slug := c.Param("slug")
	db, tenantID, ok := auth.GetScoped(c, "Plugins")
	if !ok {
		return
	}

	if !plugins.IsEmbeddedSlug(slug) {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin_not_found"})
		return
	}

	if err := upsertInstallation(db.Session(&gorm.Session{NewDB: true}), tenantID, slug, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"slug": slug, "active": true})
}

// @Summary      Desativar plugin
// @Tags         plugins
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /plugins/{slug}/deactivate [post]
func (pc *PluginController) Deactivate(c *gin.Context) {
	slug := c.Param("slug")
	db, tenantID, ok := auth.GetScoped(c, "Plugins")
	if !ok {
		return
	}

	// active=false preserva histórico/auditoria (ActivatedAt/By), preferível a DELETE.
	if err := db.Session(&gorm.Session{NewDB: true}).
		Model(&models.PluginInstallation{}).
		Where(`"tenantId" = ? AND "pluginId" = ?`, tenantID, slug).
		Update("active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate plugin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"slug": slug, "active": false})
}

// upsertInstallation cria ou reativa a linha (tenantId, pluginId).
//
// UNIQUE(tenantId, pluginId) vem de migração SQL (não de tag GORM), e a constraint existe no schema de produção mas não
// no de teste, então o código procura a linha e decide create-vs-update em vez de depender de um ON CONFLICT nomeado.
// ActivatedBy fica NULL: o userId do JWT é um int e o modelo guarda *uuid.UUID; os tipos são incompatíveis.
func upsertInstallation(db *gorm.DB, tenantID uuid.UUID, slug string, now time.Time) error {
	var existing models.PluginInstallation
	err := db.Where(`"tenantId" = ? AND "pluginId" = ?`, tenantID, slug).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]interface{}{
			"active":      true,
			"activatedAt": now,
		}).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	inst := models.PluginInstallation{
		TenantID:    tenantID,
		PluginID:    slug,
		Active:      true,
		ActivatedAt: &now,
	}
	return db.Create(&inst).Error
}
