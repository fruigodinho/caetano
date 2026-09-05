package domain

import "time"

// BalanceType representa um tipo de saldo esperado (ex: Devedor, Credor, S2C).
//
// Esta estrutura mapeia os tipos definidos na folha LEGENDA do ficheiro Excel
// de regras, permitindo validações dinâmicas baseadas em base de dados.
type BalanceType struct {
	Code             string    // Código único (ex: "D", "C", "S2C", "(R2) S2C")
	ShortDescription string    // Descrição curta (ex: "Devedor", "Credor")
	ColumnIndicator  string    // Indicador de coluna: "C" (Credor), "D" (Devedor), "2" (Ambos)
	FullDescription  string    // Descrição completa da LEGENDA
	ValidationRule   string    // Regra de validação a aplicar
	CreatedAt        time.Time // Data de criação do registo
}

// ValidationRule define as regras de validação possíveis para tipos de saldo.
type ValidationRule string

const (
	// ValidationCreditOnly indica que apenas saldos credores (≤ 0) são válidos.
	//
	// Aplica-se a tipos: C, Ca, Cc
	ValidationCreditOnly ValidationRule = "credit_only"

	// ValidationDebitOnly indica que apenas saldos devedores (≥ 0) são válidos.
	//
	// Aplica-se a tipos: D, Da, Dc
	ValidationDebitOnly ValidationRule = "debit_only"

	// ValidationBothSeparate indica que qualquer saldo é válido, representado em dois campos.
	//
	// Contas com saldo devedor somadas para campo DÉBITO.
	// Contas com saldo credor somadas para campo CRÉDITO.
	//
	// Aplica-se a tipos: S2C, (R2) S2C
	ValidationBothSeparate ValidationRule = "both_separate"

	// ValidationBothNet indica que qualquer saldo é válido, representado num único campo.
	//
	// Se representadas em campo CRÉDITO: somam-se contas credoras e subtraem-se as devedoras.
	// Se representadas em campo DÉBITO: somam-se contas devedoras e subtraem-se as credoras.
	//
	// Aplica-se a tipos: S1C, Sa1C
	ValidationBothNet ValidationRule = "both_net"

	// ValidationBothBeforeTransfer indica que qualquer saldo é válido, antes de transferências.
	//
	// Esperado saldo Devedor ou Credor ANTES de transferência para inventários/rendimentos/gastos.
	// Contas saldadas antes de apuramento de resultados.
	//
	// Aplica-se a tipos: Sc
	ValidationBothBeforeTransfer ValidationRule = "both_before_transfer"
)

// Validate verifica se um saldo é válido de acordo com a regra de validação.
//
// Parameters:
//   - balance: valor do saldo a validar
//
// Returns:
//   - bool: true se o saldo é válido, false caso contrário
func (bt *BalanceType) Validate(balance float64) bool {
	switch ValidationRule(bt.ValidationRule) {
	case ValidationCreditOnly:
		return balance <= 0
	case ValidationDebitOnly:
		return balance >= 0
	case ValidationBothSeparate, ValidationBothNet, ValidationBothBeforeTransfer:
		return true // Qualquer saldo é válido
	default:
		return false // Regra desconhecida = inválido
	}
}

// IsCredit verifica se o tipo é exclusivamente credor.
func (bt *BalanceType) IsCredit() bool {
	return bt.ValidationRule == string(ValidationCreditOnly)
}

// IsDebit verifica se o tipo é exclusivamente devedor.
func (bt *BalanceType) IsDebit() bool {
	return bt.ValidationRule == string(ValidationDebitOnly)
}

// AllowsBoth verifica se o tipo permite qualquer tipo de saldo (devedor ou credor).
func (bt *BalanceType) AllowsBoth() bool {
	rule := ValidationRule(bt.ValidationRule)
	return rule == ValidationBothSeparate ||
		rule == ValidationBothNet ||
		rule == ValidationBothBeforeTransfer
}
