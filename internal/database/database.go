package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"

	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
	"github.com/yaps-sh/yaps/internal/config"
	"github.com/yaps-sh/yaps/internal/database/sqlc"
)

type Database struct {
	db      *sql.DB
	Queries *sqlc.Queries
}

//go:embed migrations/*.sql
var migrationFS embed.FS

func New(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	conn, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, err
	}

	if err = runMigrations(ctx, conn); err != nil {
		if err = conn.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close database connection", "err", err)
		}

		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err = conn.Ping(); err != nil {
		slog.ErrorContext(ctx, "failed to ping database connection", "err", err)

		if err = conn.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close database connection", "err", err)
		}

		return nil, fmt.Errorf("failed to ping database connection: %w", err)
	}

	slog.InfoContext(ctx, "database connection established")

	return &Database{
		db:      conn,
		Queries: sqlc.New(conn),
	}, nil
}

func (db *Database) Close(ctx context.Context) error {
	if err := db.db.Close(); err != nil {
		return err
	}

	return nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrationFS)
	goose.SetLogger(gooseLogger{})

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	slog.DebugContext(ctx, "running migrations")
	return goose.Up(db, "migrations")
}

type gooseLogger struct{}

func (gooseLogger) Fatalf(format string, v ...any) {
	slog.Error("goose migration failed", "msg", fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (gooseLogger) Printf(format string, v ...any) {
	slog.Info("goose migration", "msg", fmt.Sprintf(format, v...))
}
