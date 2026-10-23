package middleware

import (
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SystemScope dá à requisição o escopo de sistema (BYPASSRLS) do banco. É restrito a rotas que não pertencem a uma
// empresa e não têm token: login, refresh, configurações públicas, setup inicial, registro e webhooks de canal (que
// descobrem a empresa pela conexão e passam a rodar no escopo dela). Um teste de arquitetura limita quem pode usar.
func SystemScope(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, done := tenantdb.SystemCtx(c.Request.Context(), db)
		defer done()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
