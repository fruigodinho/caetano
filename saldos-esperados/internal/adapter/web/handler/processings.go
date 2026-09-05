package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ProcessingsHandler struct {
	queries *postgres.Queries
}

func NewProcessingsHandler(db *sql.DB) *ProcessingsHandler {
	return &ProcessingsHandler{
		queries: postgres.New(db),
	}
}

// List exibe o histórico de uploads.
func (h *ProcessingsHandler) List(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	limit := 20
	offset := (page - 1) * limit

	uploads, err := h.queries.ListUploads(c.Request.Context(), postgres.ListUploadsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "saldos-esperados/processings_list", AddCommonData(c, gin.H{
			"Title": "Erro",
			"Error": "Erro ao carregar histórico.",
		}))
		return
	}

	// Decrypt labels
	for i := range uploads {
		if uploads[i].Label.Valid && uploads[i].Label.String != "" {
			decrypted, err := service.DecryptText(uploads[i].Label.String)
			if err == nil {
				uploads[i].Label.String = decrypted
			}
			// If error (decryption failed), keep original text (legacy support)
		}
	}

	count, _ := h.queries.CountUploads(c.Request.Context())
	totalPages := int(count) / limit
	if int(count)%limit > 0 {
		totalPages++
	}

	c.HTML(http.StatusOK, "saldos-esperados/processings_list", AddCommonData(c, gin.H{
		"Title":      "Histórico de Processamentos",
		"Uploads":    uploads,
		"Page":       page,
		"TotalPages": totalPages,
		"NextPage":   page + 1,
		"PrevPage":   page - 1,
	}))
}

// Download exporta os resultados de um upload para Excel.
func (h *ProcessingsHandler) Download(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/history")
		return
	}

	// Obter registos
	records, err := h.queries.ListProcessedRecordsByUpload(c.Request.Context(), int32(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/history?error=Erro ao carregar registos")
		return
	}

	// Criar Excel
	f := excelize.NewFile()
	sheet := "Resultados"
	f.SetSheetName("Sheet1", sheet)

	// Cabeçalhos
	headers := []string{"Conta", "Designação", "Saldo", "Regra Aplicada", "Tipo Esperado", "Resultado"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Dados
	for i, r := range records {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.AccountNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.AccountName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Balance) // Formatar decimal se necessário

		rule := r.ExpectedRuleAccount.String
		if !r.ExpectedRuleAccount.Valid {
			rule = "N/A"
		}
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), rule)

		typeCode := r.BalanceTypeCode.String
		if !r.BalanceTypeCode.Valid {
			typeCode = r.ExpectedBalanceType.String // Fallback legacy
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), typeCode)

		result := "Incorreto"
		if r.IsCorrect {
			result = "Correto"
		}
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), result)
	}

	// Configurar headers de resposta
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=processamento_%d.xlsx", id))

	if err := f.Write(c.Writer); err != nil {
		c.Status(http.StatusInternalServerError)
	}
}
