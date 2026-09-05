package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/auth"
)

// Identity resolve a identidade do pedido para um utilizador interno.
//
// Corre sempre depois de CloudflareAccess, que é quem valida o JWT e deixa o email
// no contexto. Aqui só se faz a tradução email -> utilizador.
//
// Responde 403 (e não 401) a um email desconhecido ou desativado: a autenticação já
// correu bem (é a Cloudflare que autentica, ou o bypass de dev), o que falta é
// autorização - um 401 levaria o browser a repetir uma autenticação que não
// resolveria nada.
func Identity(r *auth.Resolver, devMode bool, devAutoLoginEmail string) gin.HandlerFunc {
	devEmail := auth.NormalizeEmail(devAutoLoginEmail)
	if devMode && devEmail != "" {
		log.Printf("aviso: modo dev - todos os pedidos autenticados como %q (dev.auto_login_email)", devEmail)
	}

	return func(c *gin.Context) {
		email := EmailFromContext(c)
		if email == "" && devMode {
			// Em dev o CloudflareAccess faz bypass e não deixa email nenhum; sem
			// esta identidade explícita não haveria como testar a aplicação.
			email = devEmail
		}

		principal, err := r.Resolve(c.Request.Context(), email)
		if err != nil {
			abortForbidden(c, err, devMode)
			return
		}

		c.Set(auth.ContextKey, &auth.Context{Principal: principal})
		c.Next()
	}
}

func abortForbidden(c *gin.Context, err error, devMode bool) {
	reason := "A sua conta não tem acesso a esta aplicação."
	switch {
	case errors.Is(err, auth.ErrDisabledUser):
		reason = "A sua conta está desativada."
	case errors.Is(err, auth.ErrNoEmail):
		reason = "Não foi possível identificar o utilizador do pedido."
		if devMode {
			reason += " Em modo dev, defina dev.auto_login_email na configuração com um email existente em core_users."
		}
	case !errors.Is(err, auth.ErrUnknownUser):
		log.Printf("[auth] erro a resolver identidade: %v", err)
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	c.String(http.StatusForbidden, reason)
	c.Abort()
}

// RequireUser garante que há identidade resolvida. É a rede de segurança para um
// handler registado fora do grupo autenticado por engano.
func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := auth.From(c); !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// RequireAdmin restringe o acesso a administradores.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ac, ok := auth.From(c)
		if !ok || !ac.Principal.IsAdmin() {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
