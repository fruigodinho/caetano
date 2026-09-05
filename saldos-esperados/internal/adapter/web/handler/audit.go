package handler

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	auditService *service.AuditService
}

func NewAuditHandler(db *sql.DB) *AuditHandler {
	return &AuditHandler{
		auditService: service.NewAuditService(db),
	}
}

func (h *AuditHandler) List(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	// Get search filters
	username := c.DefaultQuery("username", "")
	action := c.DefaultQuery("action", "")
	entity := c.DefaultQuery("entity", "")

	var logs []service.AuditLogWithUser
	var count int64
	var err error

	// Use search if any filter is provided
	if username != "" || action != "" || entity != "" {
		logs, count, err = h.auditService.Search(c.Request.Context(), username, action, entity, int32(limit), int32(offset))
	} else {
		logs, count, err = h.auditService.List(c.Request.Context(), int32(limit), int32(offset))
	}

	if err != nil {
		c.HTML(http.StatusInternalServerError, "saldos-esperados/audit_list", AddCommonData(c, gin.H{
			"Title": "Audit Logs",
			"Error": "Erro ao carregar logs",
		}))
		return
	}

	totalPages := int(math.Ceil(float64(count) / float64(limit)))

	c.HTML(http.StatusOK, "saldos-esperados/audit_list", AddCommonData(c, gin.H{
		"Title":       "Audit Logs",
		"Logs":        logs,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Username":    username,
		"Action":      action,
		"Entity":      entity,
	}))
}
