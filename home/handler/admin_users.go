package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/auth"
	"github.com/fruigodinho/caetano/core/users"
	coreweb "github.com/fruigodinho/caetano/core/web"
)

// userRoles são os quatro papéis fixos do CHECK de core_users.role.
var userRoles = []string{"admin", "manager", "leader", "user"}

// AdminUsersHandler gere core_users (criar, editar, ativar/desativar). Só
// acessível a admin — protegido por core/middleware.RequireAdmin no wiring
// das rotas, não aqui.
type AdminUsersHandler struct {
	users *users.Store
}

// NewAdminUsersHandler cria o handler.
func NewAdminUsersHandler(store *users.Store) *AdminUsersHandler {
	return &AdminUsersHandler{users: store}
}

// List mostra todos os utilizadores.
func (h *AdminUsersHandler) List(c *gin.Context) {
	list, err := h.users.List(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	c.HTML(http.StatusOK, "admin_users", coreweb.PageData(c, "Utilizadores", gin.H{
		"Users": list,
	}))
}

// ShowForm mostra o formulário de criação (sem :id) ou edição (com :id).
func (h *AdminUsersHandler) ShowForm(c *gin.Context) {
	idStr := c.Param("id")

	if idStr == "" || idStr == "new" {
		c.HTML(http.StatusOK, "admin_users_form", coreweb.PageData(c, "Novo utilizador", gin.H{
			"Roles":      userRoles,
			"TargetUser": auth.User{Role: "user", Enabled: true},
		}))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	u, err := h.users.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users")
		return
	}

	c.HTML(http.StatusOK, "admin_users_form", coreweb.PageData(c, "Editar utilizador", gin.H{
		"Roles":      userRoles,
		"TargetUser": u,
		"IsEdit":     true,
	}))
}

// Save processa a criação ou atualização de um utilizador.
func (h *AdminUsersHandler) Save(c *gin.Context) {
	idStr := c.PostForm("id")
	email := c.PostForm("email")
	name := c.PostForm("name")
	role := c.PostForm("role")
	enabled := c.PostForm("enabled") == "on"

	if idStr == "" {
		h.create(c, email, name, role)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	h.update(c, id, name, role, enabled)
}

func (h *AdminUsersHandler) create(c *gin.Context, email, name, role string) {
	_, err := h.users.Create(c.Request.Context(), email, name, role)
	if err != nil {
		msg := "Erro ao criar utilizador."
		if errors.Is(err, users.ErrEmailExists) {
			msg = "Já existe um utilizador com este email."
		}
		c.HTML(http.StatusBadRequest, "admin_users_form", coreweb.PageData(c, "Novo utilizador", gin.H{
			"Roles":      userRoles,
			"TargetUser": auth.User{Email: email, Name: name, Role: role, Enabled: true},
			"Error":      msg,
		}))
		return
	}
	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *AdminUsersHandler) update(c *gin.Context, id int64, name, role string, enabled bool) {
	// RF-4: um admin não se pode desativar a si próprio (evita ficar sem
	// nenhum admin ativo por engano).
	if ac, ok := auth.From(c); ok && ac.Principal.UserID == id && !enabled {
		u, _ := h.users.GetUserByID(c.Request.Context(), id)
		c.HTML(http.StatusBadRequest, "admin_users_form", coreweb.PageData(c, "Editar utilizador", gin.H{
			"Roles":      userRoles,
			"TargetUser": u,
			"IsEdit":     true,
			"Error":      "Não pode desativar a sua própria conta.",
		}))
		return
	}

	if err := h.users.Update(c.Request.Context(), id, name, role, enabled); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, "/admin/users")
}
