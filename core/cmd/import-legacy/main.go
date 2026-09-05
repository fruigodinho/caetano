// Command import-legacy migra, pontualmente e só com leitura na origem, os
// dados de negócio da base de dados legada de saldos-esperados
// (balance_types, expected_balances, uploads, processed_records, audit_logs)
// para o schema único "caetano" (tabelas se_*).
//
// A tabela users da origem NUNCA é lida nem migrada (RF-9/D-4): a identidade
// passa a ser inteiramente gerida por core_users + Cloudflare Access. Os
// audit_logs migrados ficam com user_id NULL, porque o utilizador que os
// gerou já não existe como conceito neste sistema — preserva-se o registo
// histórico (ação, entidade, timestamp), não a atribuição.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	coreconfig "github.com/fruigodinho/caetano/core/config"
	coredb "github.com/fruigodinho/caetano/core/db"
)

func main() {
	configPath := flag.String("config", "", "caminho para a configuração YAML do destino (default: procura caetano.dev.yaml)")
	sourceDSN := flag.String("source", "", "DSN da base de dados legada de saldos-esperados (obrigatório; só leitura)")
	flag.Parse()

	if *sourceDSN == "" {
		log.Fatal("é obrigatório indicar -source (DSN da BD legada, ex.: postgres://user:pass@localhost:5433/saldos_esperados)")
	}

	ctx := context.Background()

	destPath := *configPath
	if destPath == "" {
		found, err := coreconfig.FindConfigPath("caetano.dev.yaml")
		if err != nil {
			log.Fatalf("erro ao localizar configuração de destino: %v (use -config)", err)
		}
		destPath = found
	}
	cfg, err := coreconfig.Load(destPath)
	if err != nil {
		log.Fatalf("erro ao carregar configuração %s: %v", destPath, err)
	}

	dest, err := coredb.Open(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("erro ao ligar ao destino: %v", err)
	}
	defer dest.Close()

	src, err := sql.Open("pgx", *sourceDSN)
	if err != nil {
		log.Fatalf("erro ao abrir ligação à origem: %v", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		log.Fatalf("erro ao ligar à origem: %v", err)
	}

	steps := []struct {
		name string
		fn   func(context.Context, *sql.DB, *sql.DB) (int64, error)
	}{
		{"balance_types -> se_balance_types", importBalanceTypes},
		{"expected_balances -> se_expected_balances", importExpectedBalances},
		{"uploads -> se_uploads", importUploads},
		{"processed_records -> se_processed_records", importProcessedRecords},
		{"audit_logs -> se_audit_logs (user_id fica NULL)", importAuditLogs},
	}

	for _, step := range steps {
		n, err := step.fn(ctx, src, dest)
		if err != nil {
			log.Fatalf("erro em %s: %v", step.name, err)
		}
		log.Printf("%s: %d linhas", step.name, n)
	}

	if err := report(ctx, src, dest); err != nil {
		log.Printf("aviso: erro ao gerar relatório de contagens: %v", err)
	}
}

// report imprime, por tabela, a contagem na origem e no destino, para
// confirmar visualmente o critério de aceitação de RF-11.
func report(ctx context.Context, src, dest *sql.DB) error {
	pairs := [][2]string{
		{"balance_types", "se_balance_types"},
		{"expected_balances", "se_expected_balances"},
		{"uploads", "se_uploads"},
		{"processed_records", "se_processed_records"},
		{"audit_logs", "se_audit_logs"},
	}

	fmt.Println("\nContagens (origem -> destino):")
	for _, p := range pairs {
		srcCount, err := countRows(ctx, src, p[0])
		if err != nil {
			return err
		}
		destCount, err := countRows(ctx, dest, p[1])
		if err != nil {
			return err
		}
		mark := "✓"
		if srcCount != destCount {
			mark = "✗"
		}
		fmt.Printf("  %s %-20s %6d -> %-6d %s\n", mark, p[0], srcCount, destCount, p[1])
	}
	return nil
}

func countRows(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var n int64
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n)
	return n, err
}

func importBalanceTypes(ctx context.Context, src, dest *sql.DB) (int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT code, short_description, column_indicator, full_description, validation_rule, created_at
		FROM balance_types ORDER BY code
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO se_balance_types (code, short_description, column_indicator, full_description, validation_rule, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (code) DO UPDATE SET
			short_description = EXCLUDED.short_description,
			column_indicator = EXCLUDED.column_indicator,
			full_description = EXCLUDED.full_description,
			validation_rule = EXCLUDED.validation_rule
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var n int64
	for rows.Next() {
		var code, shortDesc, colInd, fullDesc, rule string
		var createdAt any
		if err := rows.Scan(&code, &shortDesc, &colInd, &fullDesc, &rule, &createdAt); err != nil {
			return n, err
		}
		if _, err := stmt.ExecContext(ctx, code, shortDesc, colInd, fullDesc, rule, createdAt); err != nil {
			return n, fmt.Errorf("code=%s: %w", code, err)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	return n, tx.Commit()
}

func importExpectedBalances(ctx context.Context, src, dest *sql.DB) (int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT account_number, account_name, expected_balance_type, balance_type_code, created_at, updated_at
		FROM expected_balances ORDER BY account_number
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO se_expected_balances (account_number, account_name, expected_balance_type, balance_type_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account_number) DO UPDATE SET
			account_name = EXCLUDED.account_name,
			expected_balance_type = EXCLUDED.expected_balance_type,
			balance_type_code = EXCLUDED.balance_type_code,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var n int64
	for rows.Next() {
		var accNum, accName, expType string
		var balanceTypeCode sql.NullString
		var createdAt, updatedAt any
		if err := rows.Scan(&accNum, &accName, &expType, &balanceTypeCode, &createdAt, &updatedAt); err != nil {
			return n, err
		}
		if _, err := stmt.ExecContext(ctx, accNum, accName, expType, balanceTypeCode, createdAt, updatedAt); err != nil {
			return n, fmt.Errorf("account_number=%s: %w", accNum, err)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	return n, tx.Commit()
}

func importUploads(ctx context.Context, src, dest *sql.DB) (int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, filename, label, upload_date, status FROM uploads ORDER BY id
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// O id é preservado (não gerado de novo): processed_records.upload_id
	// aponta para uploads.id na origem, e essa relação tem de sobreviver à
	// migração.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO se_uploads (id, filename, label, upload_date, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var n int64
	for rows.Next() {
		var id int32
		var filename, status string
		var label sql.NullString
		var uploadDate any
		if err := rows.Scan(&id, &filename, &label, &uploadDate, &status); err != nil {
			return n, err
		}
		if _, err := stmt.ExecContext(ctx, id, filename, label, uploadDate, status); err != nil {
			return n, fmt.Errorf("upload id=%d: %w", id, err)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT setval('se_uploads_id_seq', COALESCE((SELECT MAX(id) FROM se_uploads), 1))`); err != nil {
		return n, err
	}
	return n, tx.Commit()
}

func importProcessedRecords(ctx context.Context, src, dest *sql.DB) (int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, upload_id, account_number, account_name, balance, expected_rule_account, expected_balance_type, balance_type_code, is_correct, created_at
		FROM processed_records ORDER BY id
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO se_processed_records (id, upload_id, account_number, account_name, balance, expected_rule_account, expected_balance_type, balance_type_code, is_correct, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var n int64
	for rows.Next() {
		var id, uploadID int32
		var accNum, accName, balance string
		var expRuleAccount, expBalanceType, balanceTypeCode sql.NullString
		var isCorrect bool
		var createdAt any
		if err := rows.Scan(&id, &uploadID, &accNum, &accName, &balance, &expRuleAccount, &expBalanceType, &balanceTypeCode, &isCorrect, &createdAt); err != nil {
			return n, err
		}
		if _, err := stmt.ExecContext(ctx, id, uploadID, accNum, accName, balance, expRuleAccount, expBalanceType, balanceTypeCode, isCorrect, createdAt); err != nil {
			return n, fmt.Errorf("processed_record id=%d: %w", id, err)
		}
		n++
		if n%5000 == 0 {
			log.Printf("  ... %d processed_records copiados", n)
		}
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT setval('se_processed_records_id_seq', COALESCE((SELECT MAX(id) FROM se_processed_records), 1))`); err != nil {
		return n, err
	}
	return n, tx.Commit()
}

// importAuditLogs migra o histórico de auditoria sem atribuição de
// utilizador: user_id fica sempre NULL, porque a tabela users da origem não é
// migrada (RF-9) e um core_users.id da origem não tem correspondência válida
// no destino.
func importAuditLogs(ctx context.Context, src, dest *sql.DB) (int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT action, entity, entity_id, details, ip_address, created_at
		FROM audit_logs ORDER BY id
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO se_audit_logs (user_id, action, entity, entity_id, details, ip_address, created_at)
		VALUES (NULL, $1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var n int64
	for rows.Next() {
		var action, entity string
		var entityID, details, ipAddress sql.NullString
		var createdAt any
		if err := rows.Scan(&action, &entity, &entityID, &details, &ipAddress, &createdAt); err != nil {
			return n, err
		}
		if _, err := stmt.ExecContext(ctx, action, entity, entityID, details, ipAddress, createdAt); err != nil {
			return n, err
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	return n, tx.Commit()
}
