package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/auth"
)

// RequireArea restringe um grupo de rotas à área app/areaKey. admin passa sempre,
// sem consultar a ACL (RF-6); qualquer outro role é negado por omissão até existir
// uma concessão direta a ele ou ao seu role (RF-5).
func RequireArea(store *acl.Store, app, areaKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ac, ok := auth.From(c)
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if ac.Principal.IsAdmin() {
			c.Next()
			return
		}

		allowed, err := store.HasAccess(c.Request.Context(), ac.Principal, app, areaKey)
		if err != nil {
			log.Printf("[acl] erro ao verificar acesso a %s/%s: %v", app, areaKey, err)
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
