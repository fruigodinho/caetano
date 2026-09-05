// Package users implementa auth.Querier sobre a tabela core_users.
package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fruigodinho/caetano/core/auth"
)

// ErrEmailExists indica que já existe um utilizador com o email indicado.
var ErrEmailExists = errors.New("users: email já registado")

// Store implementa auth.Querier sobre a tabela core_users.
type Store struct {
	db *sql.DB
}

// NewStore cria um Store sobre a pool de ligações dada.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetUserByEmail devolve o utilizador com o email dado (normalizado em minúsculas).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	const q = `SELECT id, email, name, role, enabled FROM core_users WHERE email = $1`
	var u auth.User
	err := s.db.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Enabled)
	return u, err
}

// GetUserByID devolve o utilizador com o id dado.
func (s *Store) GetUserByID(ctx context.Context, id int64) (auth.User, error) {
	const q = `SELECT id, email, name, role, enabled FROM core_users WHERE id = $1`
	var u auth.User
	err := s.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Enabled)
	return u, err
}

// Create insere um novo utilizador. Devolve ErrEmailExists se o email já
// estiver registado, em vez do erro genérico de violação de unicidade da BD.
func (s *Store) Create(ctx context.Context, email, name, role string) (auth.User, error) {
	email = auth.NormalizeEmail(email)
	const q = `
		INSERT INTO core_users (email, name, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, name, role, enabled
	`
	var u auth.User
	err := s.db.QueryRowContext(ctx, q, email, name, role).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Enabled)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return auth.User{}, ErrEmailExists
		}
		return auth.User{}, err
	}
	return u, nil
}

// Update grava o nome, o role e o estado (ativo/inativo) de um utilizador
// existente. O email nunca é alterado por aqui (é a chave de resolução da
// identidade Cloudflare Access).
func (s *Store) Update(ctx context.Context, id int64, name, role string, enabled bool) error {
	const q = `UPDATE core_users SET name = $2, role = $3, enabled = $4, updated_at = now() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, q, id, name, role, enabled)
	return err
}

// List devolve todos os utilizadores, ordenados por email — usado no ecrã de
// administração de ACL para escolher o destinatário de um grant.
func (s *Store) List(ctx context.Context) ([]auth.User, error) {
	const q = `SELECT id, email, name, role, enabled FROM core_users ORDER BY email`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Enabled); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
