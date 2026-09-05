package middleware

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/auth"
	"github.com/fruigodinho/caetano/core/web"
)

// BuildNav calcula, uma vez por pedido, as secções de navegação visíveis ao
// principal autenticado, a partir do catálogo de áreas de todos os módulos
// (o mesmo catalog dado a acl.Store.SyncCatalog no arranque), agrupadas por
// AreaDef.GroupLabel, mais adminGroup - uma secção que só aparece para admin
// (ex.: administração de utilizadores/ACL, que não são áreas de negócio, não
// têm concessão própria, e admin já vê tudo por definição).
//
// Publica o resultado em web.NavContextKey, para web.PageData o usar -
// nenhum handler decide a navbar por si, em nenhum módulo, e o layout
// partilhado não tem uma segunda lista de links à parte para admin.
//
// admin vê sempre o catálogo completo, sem consultar concessões (mesma regra
// de RequireArea); os restantes roles só veem as áreas concretamente
// concedidas. Isto decide só o que aparece no menu - o acesso em si continua
// só a cargo de RequireArea (e RequireAdmin) em cada rota.
func BuildNav(store *acl.Store, catalog []acl.AreaDef, adminGroup web.NavGroup) gin.HandlerFunc {
	full := toNavGroups(catalog, nil, adminGroup)

	return func(c *gin.Context) {
		ac, ok := auth.From(c)
		if !ok {
			c.Next()
			return
		}

		if ac.Principal.IsAdmin() {
			c.Set(web.NavContextKey, full)
			c.Next()
			return
		}

		allowed, err := store.AllowedAreaKeys(c.Request.Context(), ac.Principal)
		if err != nil {
			log.Printf("[nav] erro ao calcular navegação: %v", err)
			c.Set(web.NavContextKey, []web.NavGroup{})
			c.Next()
			return
		}

		c.Set(web.NavContextKey, toNavGroups(catalog, allowed, web.NavGroup{}))
		c.Next()
	}
}

// toNavGroups converte o catálogo de áreas em secções de navegação, na ordem
// em que o catálogo é dado, agrupando entradas consecutivas com o mesmo
// GroupLabel. Ignora áreas sem Href (não têm página própria) e deduplica por
// Href (várias áreas podem apontar para a mesma página, ex.: consulta e
// gestão do mesmo ecrã). allowed == nil devolve o catálogo completo, sem
// filtrar (usado para admin). adminGroup, se tiver itens, entra como última
// secção.
func toNavGroups(catalog []acl.AreaDef, allowed map[string]bool, adminGroup web.NavGroup) []web.NavGroup {
	var groups []web.NavGroup
	var currentLabel string
	var currentItems []web.NavItem
	seenHref := make(map[string]bool, len(catalog))

	flush := func() {
		if len(currentItems) > 0 {
			groups = append(groups, web.NavGroup{Label: currentLabel, Items: currentItems})
		}
	}

	for _, a := range catalog {
		if a.Href == "" || seenHref[a.Href] {
			continue
		}
		if allowed != nil && !allowed[a.App+"/"+a.Key] {
			continue
		}
		if a.GroupLabel != currentLabel {
			flush()
			currentLabel = a.GroupLabel
			currentItems = nil
		}
		seenHref[a.Href] = true
		currentItems = append(currentItems, web.NavItem{Href: a.Href, Label: a.Label, Icon: a.Icon})
	}
	flush()

	if len(adminGroup.Items) > 0 {
		groups = append(groups, adminGroup)
	}
	return groups
}
