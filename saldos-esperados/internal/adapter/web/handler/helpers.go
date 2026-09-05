package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/auth"
)

// NavLink é um item da barra de navegação do layout partilhado (core/web).
type NavLink struct {
	Href  string
	Label string
	Icon  string
}

// nav é a navegação fixa de saldos-esperados, mostrada no topbar partilhado.
// Áreas sem acesso concedido continuam visíveis no menu (RequireArea nega no
// pedido seguinte, com 403) — simplicidade preferida a filtrar o menu por ACL.
var nav = []NavLink{
	{Href: "/saldos-esperados/dashboard", Label: "Dashboard", Icon: "📊"},
	{Href: "/saldos-esperados/data", Label: "Processamento", Icon: "⬆️"},
	{Href: "/saldos-esperados/balances", Label: "Saldos Esperados", Icon: "💰"},
	{Href: "/saldos-esperados/history", Label: "Histórico", Icon: "📁"},
	{Href: "/saldos-esperados/types", Label: "Tipos de Saldo", Icon: "🏷️"},
	{Href: "/saldos-esperados/audit", Label: "Auditoria", Icon: "📋"},
}

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

// AddCommonData adiciona os dados comuns exigidos pelo layout partilhado
// "authenticated" de core/web (Nav, CurrentUser) a todos os templates.
func AddCommonData(c *gin.Context, data gin.H) gin.H {
	if data == nil {
		data = gin.H{}
	}

	data["Layout"] = "authenticated"
	data["Nav"] = nav

	if ac, ok := auth.From(c); ok {
		data["CurrentUser"] = ac.Principal
	}

	return data
}
