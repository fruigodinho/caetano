// Package handler contém os handlers próprios de home: a página de entrada e a
// administração de ACL.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/auth"
	coreweb "github.com/fruigodinho/caetano/core/web"
)

// App é a projeção de uma aplicação montada, para a página de entrada.
type App struct {
	Name  string
	Label string
	Path  string
	Icon  string
}

// LandingHandler serve a página de entrada, listando as aplicações visíveis ao
// utilizador autenticado.
type LandingHandler struct {
	acl  *acl.Store
	apps []App
}

// NewLandingHandler cria o handler com o catálogo fixo de aplicações apps.
func NewLandingHandler(store *acl.Store, apps []App) *LandingHandler {
	return &LandingHandler{acl: store, apps: apps}
}

// Show renderiza a página de entrada. admin vê sempre todas as aplicações; os
// restantes roles só veem as que têm pelo menos uma área concedida.
func (h *LandingHandler) Show(c *gin.Context) {
	ac, ok := auth.From(c)
	if !ok {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	visible := h.apps
	if !ac.Principal.IsAdmin() {
		visible = h.visibleFor(c.Request.Context(), ac.Principal)
	}

	c.HTML(http.StatusOK, "landing", coreweb.PageData(c, "Início", gin.H{
		"Apps": visible,
	}))
}

func (h *LandingHandler) visibleFor(ctx context.Context, p auth.Principal) []App {
	allowed, err := h.acl.AppsFor(ctx, p)
	if err != nil {
		return nil
	}
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}

	var out []App
	for _, app := range h.apps {
		if set[app.Name] {
			out = append(out, app)
		}
	}
	return out
}
