package handler

import (
	"database/sql"
	"net/http"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	queries *postgres.Queries
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{
		queries: postgres.New(db),
	}
}

func (h *DashboardHandler) Show(c *gin.Context) {
	// Fetch stats
	count, err := h.queries.CountUploads(c.Request.Context())
	if err != nil {
		count = 0 // Handle error gracefully
	}

	lastUpload, err := h.queries.GetLastUpload(c.Request.Context())
	lastProcessDate := "-"
	if err == nil {
		lastProcessDate = lastUpload.UploadDate.UTC().Format("02/01/2006 15:04")
	}

	c.HTML(http.StatusOK, "saldos-esperados/dashboard", AddCommonData(c, gin.H{
		"Title": "Dashboard",
		"Stats": gin.H{
			"TotalUploads": count,
			"LastProcess":  lastProcessDate,
		},
	}))
}
