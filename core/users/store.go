// Package users implementa auth.Querier sobre a tabela core_users.
package users

import (
	"context"
	"database/sql"

	"github.com/fruigodinho/caetano/core/auth"
)

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
