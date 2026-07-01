package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations runs all database migrations from the embedded filesystem.
// It traps migrate.ErrNoChange to allow the application to boot normally if no changes are needed.
func RunMigrations(databaseURL string) error {
	logger.Log.Info("Starting database migrations...")

	d, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Log.Info("Database schema is already up to date.")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Log.Info("Database migrations applied successfully!")
	return nil
}

