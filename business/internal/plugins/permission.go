package plugins

import (
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
)

// withPermission composes auth.RequirePermission (a gin middleware that
// calls c.Next() on success) with a plain handler, since
// sdk.WatinkCore.RegisterRoute only accepts one gin.HandlerFunc — not a
// chain. c.Next() here is harmless (there's nothing after it in this
// single-handler registration); what actually gates `next` is the explicit
// c.IsAborted() check.
func withPermission(resource, action string, next gin.HandlerFunc) gin.HandlerFunc {
	check := auth.RequirePermission(resource, action)
	return func(c *gin.Context) {
		check(c)
		if c.IsAborted() {
			return
		}
		next(c)
	}
}
