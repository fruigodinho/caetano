package domain

import "time"

// AccountEntry represents a row from the Balance Sheet CSV
type AccountEntry struct {
	AccountNumber string  `json:"account_number"`
	AccountName   string  `json:"account_name"`
	Balance       float64 `json:"balance"`
}

// ExpectedBalance represents a rule from the Excel file
type ExpectedBalance struct {
	AccountNumber string
	AccountName   string
	Type          string // "Devedor" or "Credor"
}

// ProcessingResult represents the validation result for an account
type ProcessingResult struct {
	Entry       AccountEntry
	MatchedRule *ExpectedBalance
	IsCorrect   bool
	ProcessedAt time.Time
}

// Upload represents a file upload record
type Upload struct {
	ID         int
	Filename   string
	UploadDate time.Time
	Status     string
}
