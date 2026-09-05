package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type BalancesHandler struct {
	queries      *postgres.Queries
	db           *sql.DB
	auditService *service.AuditService
}

func NewBalancesHandler(db *sql.DB) *BalancesHandler {
	return &BalancesHandler{
		queries:      postgres.New(db),
		db:           db,
		auditService: service.NewAuditService(db),
	}
}

// processSearchTerm converts user wildcards to SQL wildcards.
// * -> %
// ? -> _
// If no wildcards are present, it performs an exact match (no % added).
func (h *BalancesHandler) processSearchTerm(term string) string {
	if term == "" {
		return ""
	}

	// Replace user wildcards with SQL wildcards
	term = strings.ReplaceAll(term, "*", "%")
	term = strings.ReplaceAll(term, "?", "_")

	return term
}

// List exibe a lista de saldos esperados.
func (h *BalancesHandler) List(c *gin.Context) {
	// Paginação simples
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	limit := 50
	offset := (page - 1) * limit

	// Filtros
	search := c.DefaultQuery("q", "")
	typeFilter := c.DefaultQuery("type", "")

	// Process search term for smart search
	processedSearch := h.processSearchTerm(search)

	balances, err := h.queries.ListExpectedBalances(c.Request.Context(), postgres.ListExpectedBalancesParams{
		Column1: processedSearch,
		Column2: typeFilter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "saldos-esperados/balances_list", AddCommonData(c, gin.H{
			"Title": "Erro",
			"Error": "Erro ao carregar saldos esperados.",
		}))
		return
	}

	count, _ := h.queries.CountExpectedBalances(c.Request.Context(), postgres.CountExpectedBalancesParams{
		Column1: processedSearch,
		Column2: typeFilter,
	})
	totalPages := int(count) / limit
	if int(count)%limit > 0 {
		totalPages++
	}

	// Carregar tipos para o filtro
	types, _ := h.queries.ListBalanceTypes(c.Request.Context())

	c.HTML(http.StatusOK, "saldos-esperados/balances_list", AddCommonData(c, gin.H{
		"Title":      "Saldos Esperados",
		"Balances":   balances,
		"Page":       page,
		"TotalPages": totalPages,
		"NextPage":   page + 1,
		"PrevPage":   page - 1,
		"Search":     search,
		"TypeFilter": typeFilter,
		"Types":      types,
	}))
}

// ShowForm exibe o formulário.
func (h *BalancesHandler) ShowForm(c *gin.Context) {
	account := c.Param("account")

	var balance postgres.ExpectedBalance
	isEdit := false

	if account != "" {
		var err error
		balance, err = h.queries.GetExpectedBalance(c.Request.Context(), account)
		if err != nil {
			c.Redirect(http.StatusFound, "/saldos-esperados/balances")
			return
		}
		isEdit = true
	}

	// Carregar tipos para o dropdown
	types, _ := h.queries.ListBalanceTypes(c.Request.Context())

	c.HTML(http.StatusOK, "saldos-esperados/balances_form", AddCommonData(c, gin.H{
		"Title":   "Gerir Saldo Esperado",
		"Balance": balance,
		"IsEdit":  isEdit,
		"Types":   types,
	}))
}

// Save processa criação/atualização.
func (h *BalancesHandler) Save(c *gin.Context) {
	account := c.PostForm("account_number")
	name := c.PostForm("account_name")
	typeCode := c.PostForm("expected_balance_type")

	currentUserID := CurrentUserID(c)

	// Upsert logic
	_, err := h.queries.UpsertExpectedBalance(c.Request.Context(), postgres.UpsertExpectedBalanceParams{
		AccountNumber:       account,
		AccountName:         name,
		ExpectedBalanceType: typeCode,
	})

	if err != nil {
		c.HTML(http.StatusInternalServerError, "saldos-esperados/balances_form", AddCommonData(c, gin.H{
			"Title": "Erro",
			"Error": "Erro ao gravar: " + err.Error(),
		}))
		return
	}

	_ = h.auditService.Log(c.Request.Context(), currentUserID, "UPSERT", "BALANCE", account, "Upserted balance "+account, c.ClientIP())

	c.Redirect(http.StatusFound, "/saldos-esperados/balances")
}

// Delete remove um registo.
func (h *BalancesHandler) Delete(c *gin.Context) {
	account := c.Param("account")
	currentUserID := CurrentUserID(c)

	err := h.queries.DeleteExpectedBalance(c.Request.Context(), account)
	if err == nil {
		_ = h.auditService.Log(c.Request.Context(), currentUserID, "DELETE", "BALANCE", account, "Deleted balance "+account, c.ClientIP())
	}
	c.Redirect(http.StatusFound, "/saldos-esperados/balances")
}

// Import processa o upload do Excel.
func (h *BalancesHandler) Import(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/balances?error=Ficheiro inválido")
		return
	}

	f, err := file.Open()
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/balances?error=Erro ao abrir ficheiro")
		return
	}
	defer f.Close()

	xlsx, err := excelize.OpenReader(f)
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/balances?error=Formato Excel inválido")
		return
	}

	// Ler todas as contas existentes para memória para detetar as que devem ser apagadas
	// Nota: Se a tabela for muito grande, isto pode consumir muita memória.
	// Assumindo < 100k registos, é aceitável.
	existingBalances, err := h.queries.ListAllExpectedBalances(c.Request.Context())
	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/balances?error=Erro ao ler base de dados")
		return
	}

	existingMap := make(map[string]bool)
	for _, b := range existingBalances {
		existingMap[b.AccountNumber] = true
	}

	// Processar Excel
	rows, err := xlsx.GetRows("Saldos Esperados") // Assumindo nome da folha
	if err != nil {
		// Tentar ler a primeira folha se o nome falhar
		sheets := xlsx.GetSheetList()
		if len(sheets) > 0 {
			rows, err = xlsx.GetRows(sheets[0])
		}
	}

	if err != nil {
		c.Redirect(http.StatusFound, "/saldos-esperados/balances?error=Erro ao ler linhas do Excel")
		return
	}

	countImported := 0
	for i, row := range rows {
		if i == 0 {
			continue
		} // Skip header
		if len(row) < 3 {
			continue
		}

		accNum := row[0]
		accName := row[1]
		accType := row[2]

		if accNum == "" {
			continue
		}

		_, err := h.queries.UpsertExpectedBalance(c.Request.Context(), postgres.UpsertExpectedBalanceParams{
			AccountNumber:       accNum,
			AccountName:         accName,
			ExpectedBalanceType: accType,
		})
		if err == nil {
			countImported++
			// Marcar como vista (remover do mapa de existentes)
			delete(existingMap, accNum)
		}
	}

	// Apagar as que sobraram no mapa (não estavam no Excel)
	countDeleted := 0
	for accNum := range existingMap {
		err := h.queries.DeleteExpectedBalance(c.Request.Context(), accNum)
		if err == nil {
			countDeleted++
		}
	}

	currentUserID := CurrentUserID(c)
	_ = h.auditService.Log(c.Request.Context(), currentUserID, "IMPORT", "BALANCE", "", fmt.Sprintf("Imported %d, Deleted %d", countImported, countDeleted), c.ClientIP())

	msg := fmt.Sprintf("Importação concluída: %d importados/atualizados, %d removidos.", countImported, countDeleted)
	c.Redirect(http.StatusFound, "/saldos-esperados/balances?msg="+msg)
}
