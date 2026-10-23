package plugins

import (
	"context"
	"log"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/sdk"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PluginRegistry decide se um plugin está disponível para uma empresa. Não há marketplace nem licença (removidos): o
// único gate é a ALOCAÇÃO por empresa (PluginInstallations.active), ligada pelo próprio administrador da empresa.
type PluginRegistry struct {
	db *gorm.DB
}

// NewPluginRegistry monta o registro por injeção de dependência.
func NewPluginRegistry(db *gorm.DB) *PluginRegistry {
	return &PluginRegistry{db: db}
}

// GetStatus devolve o estado do plugin para a empresa: ativo se alocado, bloqueado caso contrário (inclusive em erro de
// consulta, fail-closed). Não existe o estado somente-leitura, que vinha da licença.
func (r *PluginRegistry) GetStatus(tenantID uuid.UUID, pluginSlug string) sdk.PluginStatus {
	allocated, err := r.isAllocated(tenantID, pluginSlug)
	if err != nil {
		log.Printf("[plugins] erro ao consultar alocação de %s para tenant %s: %v", pluginSlug, tenantID, err)
		return sdk.StatusBlocked
	}
	if !allocated {
		return sdk.StatusBlocked
	}
	return sdk.StatusActive
}

// isAllocated checks PluginInstallations for an active row (tenantId,
// pluginId=slug). A missing row or active=false both mean "not allocated".
func (r *PluginRegistry) isAllocated(tenantID uuid.UUID, pluginSlug string) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var count int64
	err := tenantdb.Do(context.Background(), r.db, tenantID, func(tx *gorm.DB) error {
		return tx.Model(&models.PluginInstallation{}).
			Where(`"tenantId" = ? AND "pluginId" = ? AND active = ?`, tenantID, pluginSlug, true).
			Count(&count).Error
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
