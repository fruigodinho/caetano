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

// GrantItem é uma concessão já resolvida para exibição (com o rótulo da área
// em vez da chave crua), agrupada por destinatário no template.
type GrantItem struct {
	ID        int64
	App       string
	AreaLabel string
}

// SubjectGroup junta todas as concessões de um mesmo destinatário (utilizador
// ou role) num único bloco, para a UI organizar "concessões atuais" por
// destinatário em vez de uma linha por concessão.
type SubjectGroup struct {
	SubjectType  coreacl.SubjectType
	SubjectLabel string
	Grants       []GrantItem
}

// groupGrantsBySubject agrupa concessões por destinatário (tipo + id),
// resolve o rótulo do destinatário (email do utilizador, ou o nome do role) e
// o rótulo da área (em vez da chave crua). A ordem dos grupos é a de primeira
// aparição em grants (já vem ordenado por app/área da BD).
func groupGrantsBySubject(grants []coreacl.Grant, areaLabel map[string]string, userEmail map[string]string) []SubjectGroup {
	index := make(map[string]int)
	var groups []SubjectGroup

	for _, g := range grants {
		key := string(g.SubjectType) + ":" + g.SubjectID
		label := areaLabel[g.App+":"+g.AreaKey]
		if label == "" {
			label = g.AreaKey
		}
		item := GrantItem{ID: g.ID, App: g.App, AreaLabel: label}

		if i, ok := index[key]; ok {
			groups[i].Grants = append(groups[i].Grants, item)
			continue
		}

		subjectLabel := g.SubjectID
		if g.SubjectType == coreacl.SubjectUser {
			if email, ok := userEmail[g.SubjectID]; ok {
				subjectLabel = email
			}
		}

		index[key] = len(groups)
		groups = append(groups, SubjectGroup{
			SubjectType:  g.SubjectType,
			SubjectLabel: subjectLabel,
			Grants:       []GrantItem{item},
		})
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

	areaLabel := make(map[string]string, len(areas))
	for _, a := range areas {
		areaLabel[a.App+":"+a.Key] = a.Label
	}
	userEmail := make(map[string]string, len(knownUsers))
	for _, u := range knownUsers {
		userEmail[strconv.FormatInt(u.ID, 10)] = u.Email
	}

	c.HTML(http.StatusOK, "admin_acl", coreweb.PageData(c, "Administração de ACL", gin.H{
		"AreaGroups":    groupAreasByApp(areas),
		"SubjectGroups": groupGrantsBySubject(grants, areaLabel, userEmail),
		"Users":         knownUsers,
		"Roles":         roles,
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
