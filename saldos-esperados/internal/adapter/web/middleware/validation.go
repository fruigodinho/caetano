package middleware

import (
	"errors"
	"html"
	"path/filepath"
	"regexp"
	"strings"
)

// Expressões regulares para validação
var (
	// Username: alphanumeric + underscore, 3-32 chars
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	
	// Email: RFC 5322 simplificado
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// Account Number: numeric/alphanumeric, max 20 chars
	accountNumberRegex = regexp.MustCompile(`^[a-zA-Z0-9]{1,20}$`)
	
	// Code: alphanumeric com parênteses, max 20 chars (ex: "(R2) S2C")
	codeRegex = regexp.MustCompile(`^[a-zA-Z0-9()][a-zA-Z0-9() ]{0,19}$`)
)

// SanitizeInput remove caracteres perigosos de inputs genéricos.
//
// Esta função aplica HTML escaping para prevenir XSS e remove
// null bytes que podem causar problemas de segurança.
//
// Parameters:
//   - input: string a ser sanitizada
//
// Returns:
//   - string sanitizada
func SanitizeInput(input string) string {
	// HTML escape para prevenir XSS
	sanitized := html.EscapeString(input)
	
	// Remover null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")
	
	// Trimmar espaços
	sanitized = strings.TrimSpace(sanitized)
	
	return sanitized
}

// ValidateUsername verifica se um username é válido.
//
// Regras:
//   - 3-32 caracteres
//   - Apenas letras, números e underscore
//   - Não pode estar vazio
//
// Parameters:
//   - username: username a validar
//
// Returns:
//   - error se inválido, nil se válido
func ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username não pode estar vazio")
	}
	
	if !usernameRegex.MatchString(username) {
		return errors.New("username inválido: deve ter 3-32 caracteres (letras, números, underscore)")
	}
	
	return nil
}

// ValidateEmail verifica se um email é válido.
//
// Utiliza uma versão simplificada de RFC 5322.
//
// Parameters:
//   - email: endereço de email a validar
//
// Returns:
//   - error se inválido, nil se válido
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email não pode estar vazio")
	}
	
	if !emailRegex.MatchString(email) {
		return errors.New("email inválido")
	}
	
	if len(email) > 255 {
		return errors.New("email demasiado longo (máximo 255 caracteres)")
	}
	
	return nil
}

// ValidateAccountNumber verifica se um número de conta é válido.
//
// Regras:
//   - 1-20 caracteres
//   - Apenas letras e números
//   - Não pode estar vazio
//
// Parameters:
//   - accountNumber: número de conta a validar
//
// Returns:
//   - error se inválido, nil se válido
func ValidateAccountNumber(accountNumber string) error {
	if accountNumber == "" {
		return errors.New("número de conta não pode estar vazio")
	}
	
	if !accountNumberRegex.MatchString(accountNumber) {
		return errors.New("número de conta inválido: deve ter 1-20 caracteres alfanuméricos")
	}
	
	return nil
}

// ValidateCode verifica se um código (tipo de saldo) é válido.
//
// Regras:
//   - 1-20 caracteres
//   - Letras, números, parênteses e espaços
//   - Não pode estar vazio
//
// Parameters:
//   - code: código a validar
//
// Returns:
//   - error se inválido, nil se válido
func ValidateCode(code string) error {
	if code == "" {
		return errors.New("código não pode estar vazio")
	}
	
	if !codeRegex.MatchString(code) {
		return errors.New("código inválido: deve ter 1-20 caracteres alfanuméricos")
	}
	
	return nil
}

// ValidateFilename verifica se um nome de ficheiro é seguro.
//
// Esta função previne path traversal attacks e valida extensões.
//
// Regras:
//   - Sem path separators (/, \\)
//   - Sem ".." (directory traversal)
//   - Extensão na whitelist (.csv, .xlsx, .xls)
//
// Parameters:
//   - filename: nome do ficheiro a validar
//
// Returns:
//   - error se inválido, nil se válido
func ValidateFilename(filename string) error {
	if filename == "" {
		return errors.New("nome de ficheiro não pode estar vazio")
	}
	
	// Obter apenas o nome base (remove qualquer path)
	base := filepath.Base(filename)
	
	// Verificar path traversal
	if strings.Contains(base, "..") || 
	   strings.Contains(base, "/") || 
	   strings.Contains(base, "\\") {
		return errors.New("nome de ficheiro inválido: path traversal não permitido")
	}
	
	// Verificar extensão (whitelist)
	ext := strings.ToLower(filepath.Ext(base))
	allowedExtensions := []string{".csv", ".xlsx", ".xls"}
	
	valid := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			valid = true
			break
		}
	}
	
	if !valid {
		return errors.New("tipo de ficheiro não permitido (apenas .csv, .xlsx, .xls)")
	}
	
	return nil
}

// ValidateRole verifica se uma role é válida.
//
// Roles permitidas: Administrador, Gestor, Operador
//
// Parameters:
//   - role: role a validar
//
// Returns:
//   - error se inválida, nil se válida
func ValidateRole(role string) error {
	validRoles := []string{"Administrador", "Gestor", "Operador"}
	
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	
	return errors.New("role inválida (deve ser: Administrador, Gestor ou Operador)")
}

// ValidateDescription verifica se uma descrição tem comprimento razoável.
//
// Parameters:
//   - description: descrição a validar
//   - maxLength: comprimento máximo permitido
//
// Returns:
//   - error se inválida, nil se válida
func ValidateDescription(description string, maxLength int) error {
	if len(description) > maxLength {
		return errors.New("descrição demasiado longa")
	}
	
	return nil
}

// ValidatePasswordStrength verifica se uma password cumpre requisitos de complexidade.
//
// Regras:
//   - Mínimo 10 caracteres (retrocompatibilidade)
//   - Pelo menos 1 maiúscula
//   - Pelo menos 1 minúscula
//   - Pelo menos 1 número
//   - Pelo menos 1 símbolo especial
//
// Parameters:
//   - password: password a validar
//
// Returns:
//   - error se inválida, nil se válida
func ValidatePasswordStrength(password string) error {
	if len(password) < 10 {
		return errors.New("password deve ter pelo menos 10 caracteres")
	}
	
	// Verificar maiúscula
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	if !hasUpper {
		return errors.New("password deve conter pelo menos uma letra maiúscula")
	}
	
	// Verificar minúscula
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	if !hasLower {
		return errors.New("password deve conter pelo menos uma letra minúscula")
	}
	
	// Verificar número
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	if !hasDigit {
		return errors.New("password deve conter pelo menos um número")
	}
	
	// Verificar símbolo especial
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>_\-+=\[\]\\/'~` + "`" + `]`).MatchString(password)
	if !hasSpecial {
		return errors.New("password deve conter pelo menos um símbolo especial (!@#$%^&*...)")
	}
	
	return nil
}
