package parser

import (
	"fmt"
	"strings"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/core/domain"
	"github.com/xuri/excelize/v2"
)

type ExcelParser struct{}

func NewExcelParser() *ExcelParser {
	return &ExcelParser{}
}

func (p *ExcelParser) ParseExpectedBalances(filePath string) ([]domain.ExpectedBalance, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	// Get all rows in the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var balances []domain.ExpectedBalance
	// Header is on row 1, data starts on row 2
	for i, row := range rows {
		// Skip first line (always header)
		if i == 0 {
			continue
		}

		// Skip invalid rows
		if len(row) < 3 {
			continue
		}

		// Column mapping: Col 0 = Account Number, Col 1 = Account Name, Col 2 = Expected Type
		accNum := strings.TrimSpace(row[0])
		accName := strings.TrimSpace(row[1])
		expType := strings.TrimSpace(row[2])

		if i < 5 {
			fmt.Printf("Row %d: AccNum='%s', AccName='%s', Type='%s'\n", i, accNum, accName, expType)
		}

		// Skip empty account numbers
		if accNum == "" {
			continue
		}

		// Skip rows where account number looks like a header
		accNumLower := strings.ToLower(accNum)
		if accNumLower == "conta" || accNumLower == "account" || accNumLower == "numero" {
			continue // This is a header row, skip it
		}

		balances = append(balances, domain.ExpectedBalance{
			AccountNumber: accNum,
			AccountName:   accName,
			Type:          expType,
		})
	}

	return balances, nil
}

// ParseBalanceTypes extrai tipos de saldo da folha LEGENDA do ficheiro Excel.
//
// Lê a folha "LEGENDA" e extrai informação sobre os tipos de saldo esperado,
// incluindo código, descrições e regras de validação.
//
// Parameters:
//   - filePath: caminho para o ficheiro Excel
//
// Returns:
//   - []domain.BalanceType: slice com os tipos de saldo encontrados
//   - error: erro se houver problema ao ler o ficheiro
func (p *ExcelParser) ParseBalanceTypes(filePath string) ([]domain.BalanceType, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	// Procurar folha LEGENDA
	legendSheet := ""
	for _, sheet := range f.GetSheetList() {
		if strings.ToUpper(sheet) == "LEGENDA" {
			legendSheet = sheet
			break
		}
	}

	if legendSheet == "" {
		return nil, fmt.Errorf("sheet LEGENDA not found")
	}

	// Ler linhas da folha LEGENDA
	rows, err := f.GetRows(legendSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from LEGENDA: %w", err)
	}

	var balanceTypes []domain.BalanceType

	// Estrutura esperada das linhas de tipos de saldo (linha 18-28 aproximadamente):
	// ["Saldo esperado (por conta de movimento)", "C", "C", "Credor", "Descrição completa..."]
	// ou
	// [" ", "Ca", "C", "Credor antes...", "Descrição completa..."]
	for i, row := range rows {
		// Pular linhas antes dos tipos de saldo (linhas 1-17)
		if i < 17 {
			continue
		}

		// Parar após linha 30 (tipos terminam por volta da linha 28)
		if i > 30 {
			break
		}

		// Validar estrutura da linha (precisa ter pelo menos 5 colunas)
		if len(row) < 5 {
			continue
		}

		// Coluna 1: Código (ex: "C", "D", "S2C")
		code := strings.TrimSpace(row[1])
		if code == "" {
			continue
		}

		// Ignorar linhas que não são tipos de saldo
		// (ex: linhas com "Código SNC", "AC", "ANC", etc.)
		codeLower := strings.ToLower(code)
		if codeLower == "svat" || codeLower == "ac" || codeLower == "anc" ||
			codeLower == "apc" || codeLower == "apnc" || codeLower == "pc" ||
			codeLower == "pnc" || codeLower == "rg" || codeLower == "gr" {
			continue
		}

		// Coluna 2: Indicador de coluna ("C", "D", "2")
		columnIndicator := strings.TrimSpace(row[2])
		if columnIndicator == "" {
			continue
		}

		// Coluna 3: Descrição curta
		shortDesc := strings.TrimSpace(row[3])
		if shortDesc == "" {
			continue
		}

		// Coluna 4: Descrição completa
		fullDesc := strings.TrimSpace(row[4])

		// Inferir regra de validação baseada no código e indicador de coluna
		validationRule := inferValidationRule(code, columnIndicator)

		balanceTypes = append(balanceTypes, domain.BalanceType{
			Code:             code,
			ShortDescription: shortDesc,
			ColumnIndicator:  columnIndicator,
			FullDescription:  fullDesc,
			ValidationRule:   validationRule,
		})
	}

	if len(balanceTypes) == 0 {
		return nil, fmt.Errorf("no balance types found in LEGENDA sheet")
	}

	return balanceTypes, nil
}

// inferValidationRule infere a regra de validação baseada no código e indicador.
//
// Mapeia tipos específicos para regras de validação:
// - C, Ca, Cc → credit_only
// - D, Da, Dc → debit_only
// - S2C, (R2) S2C → both_separate
// - S1C, Sa1C → both_net
// - Sc → both_before_transfer
func inferValidationRule(code, columnIndicator string) string {
	codeUpper := strings.ToUpper(code)

	// Tipos credores
	if codeUpper == "C" || codeUpper == "CA" || codeUpper == "CC" {
		return string(domain.ValidationCreditOnly)
	}

	// Tipos devedores
	if codeUpper == "D" || codeUpper == "DA" || codeUpper == "DC" {
		return string(domain.ValidationDebitOnly)
	}

	// Tipos com ambos os saldos em 2 campos
	if codeUpper == "S2C" || code == "(R2) S2C" {
		return string(domain.ValidationBothSeparate)
	}

	// Tipos com ambos os saldos em 1 campo
	if codeUpper == "S1C" || codeUpper == "SA1C" {
		return string(domain.ValidationBothNet)
	}

	// Tipo antes de transferência
	if codeUpper == "SC" {
		return string(domain.ValidationBothBeforeTransfer)
	}

	// Default: se indicador é "2", assume both_separate
	if columnIndicator == "2" {
		return string(domain.ValidationBothSeparate)
	}

	// Se indicador é "C", assume credit_only
	if columnIndicator == "C" {
		return string(domain.ValidationCreditOnly)
	}

	// Se indicador é "D", assume debit_only
	if columnIndicator == "D" {
		return string(domain.ValidationDebitOnly)
	}

	// Fallback: both_separate
	return string(domain.ValidationBothSeparate)
}
