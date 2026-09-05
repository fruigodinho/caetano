// Package db abre e configura a ligação Postgres partilhada por todos os módulos.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open abre a pool de ligações Postgres para dsn e confirma conectividade com Ping.
//
// Parameters:
//   - ctx: contexto usado no Ping inicial
//   - dsn: connection string Postgres (postgres://user:pass@host:port/db?sslmode=...)
//
// Returns:
//   - *sql.DB: pool pronta a usar (pgx/v5/stdlib, driver "pgx")
//   - error: erro de ligação ou de ping
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir ligação à base de dados: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao verificar ligação à base de dados: %w", err)
	}

	return db, nil
}
