package handler

import (
	"net/http"
	"strconv"

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
		"Areas":  areas,
		"Grants": grants,
		"Users":  knownUsers,
		"Roles":  roles,
	}))
}

// Grant processa o formulário de concessão (a um utilizador ou a um role).
func (h *AdminACLHandler) Grant(c *gin.Context) {
	subjectType := coreacl.SubjectType(c.PostForm("subject_type"))
	subjectID := c.PostForm("subject_id")
	app := c.PostForm("app")
	areaKey := c.PostForm("area_key")

	if subjectType != coreacl.SubjectUser && subjectType != coreacl.SubjectRole {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if subjectID == "" || app == "" || areaKey == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.acl.Grant(c.Request.Context(), subjectType, subjectID, app, areaKey); err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
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
