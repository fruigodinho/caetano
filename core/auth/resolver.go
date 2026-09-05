package auth

import (
	"context"
	"database/sql"
	"errors"
)

var (
	// ErrUnknownUser - email autenticado pelo Cloudflare Access mas sem utilizador
	// correspondente em core_users. Traduz-se em 403, não 401: repetir a
	// autenticação não resolveria nada.
	ErrUnknownUser = errors.New("auth: utilizador desconhecido")
	// ErrDisabledUser - utilizador existe mas está desativado.
	ErrDisabledUser = errors.New("auth: utilizador desativado")
	// ErrNoEmail - não há identidade no pedido (sem Cloudflare Access e sem
	// autologin de dev configurado).
	ErrNoEmail = errors.New("auth: pedido sem identidade")
)

// User é a projeção mínima de core_users que o Resolver consome.
type User struct {
	ID      int64
	Email   string
	Name    string
	Role    string
	Enabled bool
}

// Querier é o subconjunto do acesso a core_users que o Resolver precisa.
type Querier interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id int64) (User, error)
}

// Resolver traduz um email para um Principal.
//
// Sem cache própria de propósito: o Cloudflare Access já controla a validade da
// identidade através do exp do JWT, cuja duração é a política de sessão configurada
// na Cloudflare. Uma cache aqui reinventaria essa expiração em paralelo, com uma
// janela de atraso própria - desativar um utilizador só faria efeito depois do TTL
// expirar. Consultar core_users a cada pedido resolvido é o preço; em troca a
// desativação tem efeito imediato e não há estado para invalidar.
type Resolver struct {
	q Querier
}

// NewResolver cria um Resolver sobre o Querier dado.
func NewResolver(q Querier) *Resolver {
	return &Resolver{q: q}
}

// Resolve devolve o Principal correspondente ao email dado.
func (r *Resolver) Resolve(ctx context.Context, email string) (Principal, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return Principal{}, ErrNoEmail
	}
	return r.load(ctx, email)
}

func (r *Resolver) load(ctx context.Context, email string) (Principal, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return Principal{}, ErrUnknownUser
	}
	if err != nil {
		return Principal{}, err
	}
	if !u.Enabled {
		return Principal{}, ErrDisabledUser
	}
	return Principal{
		UserID: u.ID,
		Email:  u.Email,
		Name:   u.Name,
		Role:   Role(u.Role),
	}, nil
}
