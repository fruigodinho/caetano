package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate aplica (up) ou reverte (down) as migrações do schema caetano contra dsn.
//
// Parameters:
//   - dsn: connection string Postgres
//   - direction: "up" ou "down"
//
// Returns:
//   - error: erro de ligação, de migração, ou direção inválida
func Migrate(dsn, direction string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("erro ao carregar migrações embutidas: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("erro ao inicializar migrador: %w", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		return fmt.Errorf("direção de migração inválida: %q (esperado up|down)", direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("erro ao aplicar migrações (%s): %w", direction, err)
	}
	return nil
}
