package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

// Migrator applies goose migrations. It must connect as the schema owner,
// not the application role.
type Migrator struct {
	db       *sql.DB
	provider *goose.Provider
}

// NewMigrator opens a dedicated connection to url for migrations in fsys.
func NewMigrator(url string, fsys fs.FS) (*Migrator, error) {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open migration connection: %w", err)
	}
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create migration provider: %w", err)
	}
	return &Migrator{db: sqlDB, provider: p}, nil
}

// Close closes the migration connection.
func (m *Migrator) Close() error { return m.db.Close() }

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context, logger *slog.Logger) error {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.InfoContext(ctx, "migration applied",
			"version", r.Source.Version, "duration_ms", r.Duration.Milliseconds())
	}
	return nil
}

// DownTo rolls back migrations newer than version.
func (m *Migrator) DownTo(ctx context.Context, version int64) error {
	if _, err := m.provider.DownTo(ctx, version); err != nil {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}

// Status lists every known migration and whether it is applied.
func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	st, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration status: %w", err)
	}
	return st, nil
}
