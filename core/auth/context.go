// Package auth define a identidade resolvida a partir do Cloudflare Access e o
// contrato mínimo de acesso a dados que essa resolução precisa.
package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextKey é a chave usada em gin.Context para publicar o *Context do pedido.
const ContextKey = "caetano_auth_context"

// Role identifica o papel do utilizador. admin tem sempre acesso total; os
// restantes são negados por omissão, salvo concessão explícita na ACL.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleLeader  Role = "leader"
	RoleUser    Role = "user"
)

// Principal é a identidade real resolvida a partir do email autenticado pelo
// Cloudflare Access (ou, em dev, pelo DevAutoLoginEmail da configuração).
type Principal struct {
	UserID int64
	Email  string
	Name   string
	Role   Role
}

// IsAdmin indica se o principal tem acesso total, sem consulta à ACL.
func (p Principal) IsAdmin() bool {
	return p.Role == RoleAdmin
}

// Context é o resultado da resolução de identidade publicado no gin.Context.
type Context struct {
	Principal Principal
}

// From lê o *Context publicado por middleware.Identity no pedido corrente.
func From(c *gin.Context) (*Context, bool) {
	v, ok := c.Get(ContextKey)
	if !ok {
		return nil, false
	}
	ac, ok := v.(*Context)
	return ac, ok
}

// NormalizeEmail normaliza um email para comparação/armazenamento (minúsculas, sem
// espaços nas extremidades).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
