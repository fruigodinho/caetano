package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/core/domain"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type CSVParser struct{}

func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// ExtractLabel extrai a primeira linha do CSV para usar como label de identificação.
//
// Parameters:
//   - filePath: caminho do ficheiro CSV
//
// Returns:
//   - string: primeira linha do CSV (label)
//   - error: erro se falhar a leitura
func (p *CSVParser) ExtractLabel(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open csv file: %w", err)
	}
	defer f.Close()

	// Handle ISO-8859-1 encoding
	reader := csv.NewReader(transform.NewReader(f, charmap.ISO8859_1.NewDecoder()))
	reader.Comma = ';'
	reader.LazyQuotes = true

	// Read first line
	firstLine, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read first line: %w", err)
	}

	// Join all columns with semicolon to preserve original format
	label := strings.Join(firstLine, ";")

	// Limit length to avoid database issues
	if len(label) > 500 {
		label = label[:500]
	}

	return label, nil
}

func (p *CSVParser) ParseBalanceSheet(filePath string, batchSize int, processBatch func([]domain.AccountEntry) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open csv file: %w", err)
	}
	defer f.Close()

	// Handle ISO-8859-1 encoding
	reader := csv.NewReader(transform.NewReader(f, charmap.ISO8859_1.NewDecoder()))
	reader.Comma = ';' // Semicolon separator
	reader.LazyQuotes = true

	// Skip first 3 lines (metadata)
	for i := 0; i < 3; i++ {
		_, err := reader.Read()
		if err != nil {
			return fmt.Errorf("failed to skip header lines: %w", err)
		}
	}

	// Read header line (line 4)
	_, err = reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header line: %w", err)
	}

	var batch []domain.AccountEntry
	lineCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading csv line %d: %w", lineCount+5, err)
		}

		// Column mapping: 0=NumConta, 1=Conta, 2=Saldo
		if len(record) < 3 {
			continue
		}

		accNum := strings.TrimSpace(record[0])
		accName := strings.TrimSpace(record[1])
		balanceStr := strings.TrimSpace(record[2])

		if lineCount < 5 {
			fmt.Printf("CSV Line %d: AccNum='%s', AccName='%s', Balance='%s'\n", lineCount+5, accNum, accName, balanceStr)
		}

		// Parse balance (European format: 1.234,56 -> 1234.56)
		balanceStr = strings.ReplaceAll(balanceStr, ".", "")
		balanceStr = strings.ReplaceAll(balanceStr, ",", ".")
		balance, err := strconv.ParseFloat(balanceStr, 64)
		if err != nil {
			// Log error but continue? Or fail? For now, skip invalid balances
			continue
		}

		// Filter only 8-digit accounts
		if len(accNum) == 8 {
			batch = append(batch, domain.AccountEntry{
				AccountNumber: accNum,
				AccountName:   accName,
				Balance:       balance,
			})
		}

		if len(batch) >= batchSize {
			if err := processBatch(batch); err != nil {
				return fmt.Errorf("failed to process batch: %w", err)
			}
			batch = nil // Reset batch
		}
		lineCount++
	}

	// Process remaining items
	if len(batch) > 0 {
		if err := processBatch(batch); err != nil {
			return fmt.Errorf("failed to process final batch: %w", err)
		}
	}

	return nil
}
