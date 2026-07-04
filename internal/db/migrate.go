package db

import (
	"context"
	"fmt"
	"log/slog"

	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/consumer/*.sql
var migrationsFS embed.FS

// MigrateConsumerDB runs goose migrations on the given consumer pool.
func MigrateConsumerDB(ctx context.Context, pool *pgxpool.Pool) error {
	_ = ctx
	stdlibDB := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := stdlibDB.Close(); err != nil {
			slog.Error("failed to close migration db", "error", err)
		}
	}()

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(stdlibDB, "migrations/consumer"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
