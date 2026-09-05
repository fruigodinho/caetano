package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/auth"
	coreweb "github.com/fruigodinho/caetano/core/web"
)

// CurrentUserID devolve o id (core_users.id) do utilizador autenticado no pedido.
// A identidade é resolvida pelo core (Cloudflare Access / autologin de dev), nunca
// por uma sessão própria deste módulo.
func CurrentUserID(c *gin.Context) int32 {
	ac, ok := auth.From(c)
	if !ok {
		return 0
	}
	return int32(ac.Principal.UserID)
}

// AddCommonData adiciona os dados exigidos pelo layout partilhado
// "authenticated" (Layout, Title, Nav, CurrentUser) a um template. Delega em
// core/web.PageData - a mesma função usada por home e xmldri - para não haver
// três cópias divergentes desta lógica; Nav já não é uma lista escrita à mão
// neste módulo, vem do catálogo de core/middleware.BuildNav.
func AddCommonData(c *gin.Context, data gin.H) gin.H {
	if data == nil {
		data = gin.H{}
	}
	title, _ := data["Title"].(string)
	delete(data, "Title")
	return coreweb.PageData(c, title, data)
}
