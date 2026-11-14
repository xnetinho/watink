package auth

import (
	"errors"
	"net/http"

	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantUUIDFromContext extrai o ID do tenant do contexto Gin de forma segura.
func TenantUUIDFromContext(c *gin.Context) (uuid.UUID, error) {
	v, ok := c.Get("tenantId")
	if !ok || v == nil {
		return uuid.Nil, errors.New("tenantId not found")
	}
	switch t := v.(type) {
	case uuid.UUID:
		return t, nil
	case string:
		id, err := uuid.Parse(t)
		if err != nil {
			return uuid.Nil, err
		}
		return id, nil
	default:
		return uuid.Nil, errors.New("invalid tenantId type")
	}
}

// GetDB extracts the GORM DB from Gin context, with fail-fast panic if missing.
// The DB must be injected by the IsAuth middleware.
func GetDB(c *gin.Context) *gorm.DB {
	if db, ok := c.Get("db"); ok {
		return db.(*gorm.DB)
	}
	// Fail-fast: o middleware IsAuth DEVE injetar o db no contexto
	panic("DB NOT INJECTED INTO CONTEXT - IsAuth MISSING")
}

// GetScopedDB applies table-specific scoping rules to the database context.
// It adds tenant scoping and optionally user-specific filtering based on alcance.
func GetScopedDB(c *gin.Context, table string) *gorm.DB {
	db := GetDB(c)
	alcance, _ := c.Get("alcance")
	userID, _ := c.Get("userId")
	tenantID, err := TenantUUIDFromContext(c)
	if err != nil {
		// If tenantUUIDFromContext fails, the request should have been rejected upstream.
		// Fail-closed: return nil tenant scope which will produce no results.
		tenantID = uuid.Nil
	}

	// alcance "tenant" (Gerente Geral/Administrador) e "plataforma" (superadmin)
	// enxergam tudo dentro do tenant.
	if alcance == "tenant" || alcance == "plataforma" {
		return db.Where("\"tenantId\" = ?", tenantID)
	}

	// Visibilidade do operador (Q4): os tickets dele + os SEM DONO das filas
	// dele. Ticket de outro operador nunca aparece, nem na mesma fila.
	// NOTE: join tables user_queues / channel_connection_queues usam snake_case.
	switch table {
	case "Tickets":
		return db.Where(
			"\"tenantId\" = ? AND "+TicketVisibilitySQL(""),
			tenantID, userID, userID, userID)
	case "Contacts":
		return db.Where(
			"\"tenantId\" = ? AND ( "+
				"\"walletUserId\" = ? "+
				"OR id IN (SELECT \"contactId\" FROM \"Tickets\" WHERE \"tenantId\" = ? AND "+TicketVisibilitySQL("")+") "+
				")",
			tenantID, userID, tenantID, userID, userID, userID)
	default:
		return db.Where("\"tenantId\" = ?", tenantID)
	}
}

// TicketVisibilitySQL é a regra única de visibilidade do ticket para quem NÃO
// tem alcance de empresa (Q4, PR 2b): atribuído ao usuário, ou sem dono e numa
// fila/conexão dele. Três placeholders, todos o userId.
//
// prefix é o alias da tabela de Tickets na consulta externa (ex.: "t.").
func TicketVisibilitySQL(prefix string) string {
	return `(` + prefix + `"userId" = ? OR (` + prefix + `"userId" IS NULL AND (` +
		prefix + `"queueId" IN (SELECT queue_id FROM user_queues WHERE user_id = ?) ` +
		`OR ` + prefix + `"channelConnectionId" IN (SELECT wq.channel_connection_id FROM channel_connection_queues wq WHERE wq.queue_id IN (SELECT queue_id FROM user_queues WHERE user_id = ?)))))`
}

// GetScoped extracts the tenant-scoped DB (filtro "tenantId" explícito, ADR 0001) and the validated tenantID.
// If tenantID is missing or invalid, it responds with a safe error and returns ok=false.
// Usage: db, tenantID, ok := auth.GetScoped(c, "Tickets"); if !ok { return }
func GetScoped(c *gin.Context, table string) (*gorm.DB, uuid.UUID, bool) {
	tenantID, err := TenantUUIDFromContext(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err, "Invalid tenant context")
		return nil, uuid.Nil, false
	}

	db := GetScopedDB(c, table)
	// GetScopedDB usa GetDB, que dá panic se o IsAuth não rodou: fail-fast é o certo (rota sem autenticação).
	return db, tenantID, true
}
