package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/core/domain"
)

// Validator é responsável pela validação de saldos de contas.
type Validator struct {
	queries *postgres.Queries
	db      *sql.DB
}

// NewValidator cria uma nova instância do Validator.
//
// Parameters:
//   - db: conexão com a base de dados
//
// Returns:
//   - *Validator: nova instância do validador
func NewValidator(db *sql.DB) *Validator {
	return &Validator{
		queries: postgres.New(db),
		db:      db,
	}
}

// ImportBalanceTypes importa os tipos de saldo da folha LEGENDA para a base de dados.
//
// Parameters:
//   - ctx: contexto da execução
//   - balanceTypes: lista de tipos de saldo a importar
//
// Returns:
//   - error: erro se a importação falhar
func (v *Validator) ImportBalanceTypes(ctx context.Context, balanceTypes []domain.BalanceType) error {
	tx, err := v.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := v.queries.WithTx(tx)

	// Limpar tipos existentes
	if err := qtx.DeleteAllBalanceTypes(ctx); err != nil {
		return fmt.Errorf("failed to clear existing balance types: %w", err)
	}

	// Inserir novos tipos
	for _, bt := range balanceTypes {
		if err := qtx.CreateBalanceType(ctx, postgres.CreateBalanceTypeParams{
			Code:             bt.Code,
			ShortDescription: bt.ShortDescription,
			ColumnIndicator:  bt.ColumnIndicator,
			FullDescription:  bt.FullDescription,
			ValidationRule:   bt.ValidationRule,
		}); err != nil {
			return fmt.Errorf("failed to insert balance type %s: %w", bt.Code, err)
		}
	}

	return tx.Commit()
}

// ImportRules importa as regras de saldos esperados para a base de dados.
//
// Parameters:
//   - ctx: contexto da execução
//   - rules: lista de regras a importar
//
// Returns:
//   - error: erro se a importação falhar
func (v *Validator) ImportRules(ctx context.Context, rules []domain.ExpectedBalance) error {
	tx, err := v.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := v.queries.WithTx(tx)

	// Clear existing rules? Or update? For now, clear all to be safe as per requirements implied
	if err := qtx.DeleteAllExpectedBalances(ctx); err != nil {
		return fmt.Errorf("failed to clear existing rules: %w", err)
	}

	for _, rule := range rules {
		_, err := qtx.UpsertExpectedBalance(ctx, postgres.UpsertExpectedBalanceParams{
			AccountNumber:       rule.AccountNumber,
			AccountName:         rule.AccountName,
			ExpectedBalanceType: rule.Type,
		})
		if err != nil {
			return fmt.Errorf("failed to insert rule for account %s: %w", rule.AccountNumber, err)
		}
	}

	return tx.Commit()
}

// ProcessBatch processa um lote de entradas de conta e valida os saldos.
//
// Parameters:
//   - ctx: contexto da execução
//   - uploadID: ID do upload associado
//   - entries: lista de entradas de conta a processar
//
// Returns:
//   - error: erro se o processamento falhar
func (v *Validator) ProcessBatch(ctx context.Context, uploadID int32, entries []domain.AccountEntry) error {
	// Pre-load all rules to memory for performance? Or query one by one?
	// Given the number of accounts might be large, querying one by one might be slow.
	// But for simplicity in Phase 1, let's query. Optimization: Load all rules into a map.

	// Load all rules
	rulesDB, err := v.queries.ListAllExpectedBalances(ctx)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	rulesMap := make(map[string]postgres.ExpectedBalance)
	for _, r := range rulesDB {
		rulesMap[r.AccountNumber] = r
	}

	tx, err := v.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := v.queries.WithTx(tx)

	for _, entry := range entries {
		matchedRule, ruleAcc := v.findRule(entry.AccountNumber, rulesMap)
		isCorrect := true
		ruleAccNum := ""
		balanceType := ""

		if matchedRule != nil {
			ruleAccNum = ruleAcc
			balanceType = matchedRule.ExpectedBalanceType
			isCorrect = v.validateBalance(entry.Balance, matchedRule.ExpectedBalanceType)
		} else {
			// No rule found: marca como correto mas sem tipo definido
			// O campo expected_balance_type ficará NULL na base de dados
			isCorrect = true
		}

		if i := 0; i < 5 && len(entries) > 0 { // Just log a few
			// fmt.Printf("Inserting: Acc=%s, ExpectedRule=%s, Type=%s\n", entry.AccountNumber, ruleAccNum, balanceType)
		}

		_, err := qtx.CreateProcessedRecord(ctx, postgres.CreateProcessedRecordParams{
			UploadID:            int32(uploadID),
			AccountNumber:       entry.AccountNumber,
			AccountName:         entry.AccountName,
			Balance:             fmt.Sprintf("%.2f", entry.Balance),
			ExpectedRuleAccount: sql.NullString{String: ruleAccNum, Valid: ruleAccNum != ""},
			ExpectedBalanceType: sql.NullString{String: balanceType, Valid: balanceType != ""},
			IsCorrect:           isCorrect,
		})
		if err != nil {
			return fmt.Errorf("failed to save record %s: %w", entry.AccountNumber, err)
		}
	}

	return tx.Commit()
}

// CreateUploadRecord cria um registo de upload na base de dados.
//
// Parameters:
//   - ctx: contexto da execução
//   - filename: nome do ficheiro enviado
//   - label: label de identificação (primeira linha do CSV)
//
// Returns:
//   - int32: ID do upload criado
//   - error: erro se a criação falhar
func (v *Validator) CreateUploadRecord(ctx context.Context, filename, label string) (int32, error) {
	upload, err := v.queries.CreateUpload(ctx, postgres.CreateUploadParams{
		Filename: filename,
		Label:    sql.NullString{String: label, Valid: label != ""},
		Status:   "processing",
	})
	if err != nil {
		return 0, err
	}
	return int32(upload.ID), nil
}

// UpdateUploadStatus atualiza o estado de um upload.
//
// Parameters:
//   - ctx: contexto da execução
//   - uploadID: ID do upload
//   - status: novo estado ("processed", "failed")
//
// Returns:
//   - error: erro se a atualização falhar
func (v *Validator) UpdateUploadStatus(ctx context.Context, uploadID int32, status string) error {
	return v.queries.UpdateUploadStatus(ctx, postgres.UpdateUploadStatusParams{
		ID:     int32(uploadID),
		Status: status,
	})
}

// findRule finds the matching rule for an account, checking parent accounts if necessary
func (v *Validator) findRule(accNum string, rules map[string]postgres.ExpectedBalance) (*postgres.ExpectedBalance, string) {
	// Try exact match
	if rule, ok := rules[accNum]; ok {
		return &rule, accNum
	}

	// Try parent accounts (remove digits from right)
	for i := len(accNum) - 1; i > 0; i-- {
		parentNum := accNum[:i]
		if rule, ok := rules[parentNum]; ok {
			return &rule, parentNum
		}
	}

	return nil, ""
}

// validateBalance valida se um saldo está correto de acordo com o tipo esperado.
//
// A validação consulta a tabela balance_types para obter a regra de validação.
// Se o tipo não existir na BD, tenta validação via lógica hardcoded (fallback).
//
// Parameters:
//   - balance: valor do saldo a validar
//   - expectedType: código do tipo esperado (ex: "D", "C", "S2C")
//
// Returns:
//   - bool: true se o saldo é válido, false caso contrário
func (v *Validator) validateBalance(balance float64, expectedType string) bool {
	expectedType = strings.TrimSpace(expectedType)

	// Tentar consultar balance_types
	balanceType, err := v.queries.GetBalanceType(context.Background(), expectedType)
	if err == nil {
		// Tipo encontrado na BD - usar regra de validação
		bt := domain.BalanceType{
			Code:           balanceType.Code,
			ValidationRule: balanceType.ValidationRule,
		}
		return bt.Validate(balance)
	}

	// Fallback: tipo não encontrado na BD, usar lógica hardcoded
	log.Printf("Balance type '%s' not found in database, using fallback validation", expectedType)

	switch expectedType {
	case "D", "Da", "Dc":
		return balance >= 0
	case "C", "Ca", "Cc":
		return balance <= 0
	case "S2C", "(R2) S2C", "S1C", "Sa1C", "Sc":
		return true
	default:
		// Tipos legados
		expectedTypeLower := strings.ToLower(expectedType)
		switch expectedTypeLower {
		case "positivo", "devedor":
			return balance >= 0
		case "negativo", "credor":
			return balance <= 0
		default:
			log.Printf("Unknown expected type: %s", expectedType)
			return true // Assume válido para evitar falsos negativos
		}
	}
}
