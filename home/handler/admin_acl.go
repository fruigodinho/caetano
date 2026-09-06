package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	coreacl "github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/users"
	coreweb "github.com/fruigodinho/caetano/core/web"
)

// AdminACLHandler gere o catálogo de áreas de ACL e as suas concessões
// (utilizador ou role x aplicação x área). Só acessível a admin — protegido por
// core/middleware.RequireAdmin no wiring das rotas.
type AdminACLHandler struct {
	acl   *coreacl.Store
	users *users.Store
}

// NewAdminACLHandler cria o handler.
func NewAdminACLHandler(acl *coreacl.Store, users *users.Store) *AdminACLHandler {
	return &AdminACLHandler{acl: acl, users: users}
}

// roles são os quatro papéis fixos do sistema; admin não precisa de concessões
// (RF-6), mas listar não faz mal — pode ser útil documentar explicitamente.
var roles = []string{"manager", "leader", "user"}

// AreaGroup agrupa as áreas ativas de uma aplicação, para o template desenhar
// um bloco de checkboxes por aplicação.
type AreaGroup struct {
	App   string
	Areas []coreacl.Area
}

// groupAreasByApp agrupa areas (já ordenadas por app, area_key por
// core/acl.Store.ListActiveAreas) em blocos consecutivos por app.
func groupAreasByApp(areas []coreacl.Area) []AreaGroup {
	var groups []AreaGroup
	for _, a := range areas {
		if n := len(groups); n > 0 && groups[n-1].App == a.App {
			groups[n-1].Areas = append(groups[n-1].Areas, a)
			continue
		}
		groups = append(groups, AreaGroup{App: a.App, Areas: []coreacl.Area{a}})
	}
	return groups
}

// Show lista as áreas ativas, as concessões existentes e os utilizadores
// conhecidos (para o formulário de concessão).
func (h *AdminACLHandler) Show(c *gin.Context) {
	ctx := c.Request.Context()

	areas, err := h.acl.ListActiveAreas(ctx)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	grants, err := h.acl.ListGrants(ctx)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	knownUsers, err := h.users.List(ctx)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	c.HTML(http.StatusOK, "admin_acl", coreweb.PageData(c, "Administração de ACL", gin.H{
		"AreaGroups": groupAreasByApp(areas),
		"Grants":     grants,
		"Users":      knownUsers,
		"Roles":      roles,
	}))
}

// Grant processa o formulário de concessão (a um utilizador ou a um role):
// cada área marcada chega como "app:area_key" em area_key[], para conceder
// várias áreas numa só submissão.
func (h *AdminACLHandler) Grant(c *gin.Context) {
	ctx := c.Request.Context()

	subjectType := coreacl.SubjectType(c.PostForm("subject_type"))
	subjectID := c.PostForm("subject_id")
	rawAreas := c.PostFormArray("area_key")

	if subjectType != coreacl.SubjectUser && subjectType != coreacl.SubjectRole {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if subjectID == "" || len(rawAreas) == 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	type pair struct{ app, key string }
	pairs := make([]pair, 0, len(rawAreas))
	for _, raw := range rawAreas {
		app, key, ok := strings.Cut(raw, ":")
		if !ok || app == "" || key == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		pairs = append(pairs, pair{app, key})
	}

	activeAreas, err := h.acl.ListActiveAreas(ctx)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	active := make(map[string]bool, len(activeAreas))
	for _, a := range activeAreas {
		active[a.App+":"+a.Key] = true
	}
	for _, p := range pairs {
		if !active[p.app+":"+p.key] {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}

	for _, p := range pairs {
		if err := h.acl.Grant(ctx, subjectType, subjectID, p.app, p.key); err != nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin/acl")
}

// Revoke remove uma concessão pelo seu id.
func (h *AdminACLHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.acl.Revoke(c.Request.Context(), id); err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	c.Redirect(http.StatusFound, "/admin/acl")
}
