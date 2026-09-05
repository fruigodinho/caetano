package handler

import (
	"database/sql"
	"net/http"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
	"github.com/gin-gonic/gin"
)

type TypesHandler struct {
	queries      *postgres.Queries
	auditService *service.AuditService
}

func NewTypesHandler(db *sql.DB) *TypesHandler {
	return &TypesHandler{
		queries:      postgres.New(db),
		auditService: service.NewAuditService(db),
	}
}

// List exibe a lista de tipos de saldo.
func (h *TypesHandler) List(c *gin.Context) {
	types, err := h.queries.ListBalanceTypes(c.Request.Context())
	if err != nil {
	c.HTML(http.StatusInternalServerError, "saldos-esperados/types_list", AddCommonData(c, gin.H{
		"Title": "Erro - Saldos Esperados",
		"Error": "Erro ao carregar tipos de saldo.",
	}))
		return
	}

	c.HTML(http.StatusOK, "saldos-esperados/types_list", AddCommonData(c, gin.H{
		"Title": "Tipos de Saldo",
		"Types": types,
	}))
}

// ShowForm exibe o formulário de criação ou edição.
func (h *TypesHandler) ShowForm(c *gin.Context) {
	code := c.Param("code")

	var balanceType postgres.BalanceType
	isEdit := false

	if code != "" {
		var err error
		balanceType, err = h.queries.GetBalanceType(c.Request.Context(), code)
		if err != nil {
			c.Redirect(http.StatusFound, "/saldos-esperados/types")
			return
		}
		isEdit = true
	}

	c.HTML(http.StatusOK, "saldos-esperados/types_form", AddCommonData(c, gin.H{
		"Title":       "Gerir Tipo de Saldo",
		"Type":        balanceType,
		"IsEdit":      isEdit,
		"Description": "Definição de regras para validação de saldos.",
	}))
}

// Save processa a criação ou atualização de um tipo.
func (h *TypesHandler) Save(c *gin.Context) {
	code := c.PostForm("code")
	shortDesc := c.PostForm("short_description")
	colInd := c.PostForm("column_indicator")
	fullDesc := c.PostForm("full_description")
	rule := c.PostForm("validation_rule")
	isEdit := c.PostForm("is_edit") == "true"

	currentUserID := CurrentUserID(c)

	var err error
	if isEdit {
		_, err = h.queries.UpdateBalanceType(c.Request.Context(), postgres.UpdateBalanceTypeParams{
			Code:             code,
			ShortDescription: shortDesc,
			ColumnIndicator:  colInd,
			FullDescription:  fullDesc,
			ValidationRule:   rule,
		})
		if err == nil {
			_ = h.auditService.Log(c.Request.Context(), currentUserID, "UPDATE", "TYPE", code, "Updated type "+code, c.ClientIP())
		}
	} else {
		err = h.queries.CreateBalanceType(c.Request.Context(), postgres.CreateBalanceTypeParams{
			Code:             code,
			ShortDescription: shortDesc,
			ColumnIndicator:  colInd,
			FullDescription:  fullDesc,
			ValidationRule:   rule,
		})
		if err == nil {
			_ = h.auditService.Log(c.Request.Context(), currentUserID, "CREATE", "TYPE", code, "Created type "+code, c.ClientIP())
		}
	}

	if err != nil {
		// Em caso de erro, voltar ao formulário com mensagem (simplificado por agora)
		c.HTML(http.StatusInternalServerError, "saldos-esperados/types_form", AddCommonData(c, gin.H{
			"Title":  "Erro",
			"Error":  "Erro ao gravar tipo: " + err.Error(),
			"Type":   postgres.BalanceType{Code: code, ShortDescription: shortDesc, ColumnIndicator: colInd, FullDescription: fullDesc, ValidationRule: rule},
			"IsEdit": isEdit,
		}))
		return
	}

	c.Redirect(http.StatusFound, "/saldos-esperados/types")
}

// Delete remove um tipo de saldo.
func (h *TypesHandler) Delete(c *gin.Context) {
	code := c.Param("code")
	currentUserID := CurrentUserID(c)

	err := h.queries.DeleteBalanceType(c.Request.Context(), code)
	if err == nil {
		_ = h.auditService.Log(c.Request.Context(), currentUserID, "DELETE", "TYPE", code, "Deleted type "+code, c.ClientIP())
	}
	c.Redirect(http.StatusFound, "/saldos-esperados/types")
}
