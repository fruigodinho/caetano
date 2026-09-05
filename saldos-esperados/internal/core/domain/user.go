package domain

// User representa um utilizador do sistema.
type User struct {
	ID       int
	Username string
	Password string // Hashed
	Secret   string // TOTP Secret
}
