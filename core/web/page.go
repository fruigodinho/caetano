package web

import (
	"maps"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/auth"
)

// NavContextKey é a chave em gin.Context onde core/middleware.BuildNav publica
// os itens de navegação calculados para o pedido corrente.
const NavContextKey = "caetano_nav"

// NavItem é um item da barra de navegação partilhada.
type NavItem struct {
	Href  string
	Label string
	Icon  string
}

// NavGroup é uma secção da barra de navegação (ex.: "xmldri", "Saldos
// Esperados", "Administração"), com um cabeçalho e os itens dentro dela.
type NavGroup struct {
	Label string
	Items []NavItem
}

// PageData constrói o envelope comum a qualquer página do layout partilhado
// "authenticated" (Layout, Title, Nav, CurrentUser), fundido com extra.
//
// É a ÚNICA função em todo o caetano que deve preencher Nav ou CurrentUser -
// nenhum handler, em nenhum módulo, constrói estes campos à mão. Nav vem do
// contexto do pedido (publicado uma vez por core/middleware.BuildNav, a
// partir do catálogo de áreas de todos os módulos, filtrado pelo que o
// utilizador pode ver); CurrentUser vem de auth.From. Antes desta função
// existir, cada módulo tinha a sua própria cópia desta lógica (e uma delas
// tinha uma lista de nav escrita à mão, divergente das rotas reais) - nunca
// mais.
func PageData(c *gin.Context, title string, extra gin.H) gin.H {
	data := gin.H{}
	maps.Copy(data, extra)

	data["Layout"] = "authenticated"
	data["Title"] = title

	if nav, ok := c.Get(NavContextKey); ok {
		data["Nav"] = nav
	} else {
		data["Nav"] = []NavGroup{}
	}

	if ac, ok := auth.From(c); ok {
		data["CurrentUser"] = ac.Principal
	}

	return data
}
