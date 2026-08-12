package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/yaps-sh/yaps/internal/config"
	"github.com/yaps-sh/yaps/internal/database/sqlc"
	_ "modernc.org/sqlite"
)

type Database struct {
	WriteQueries *sqlc.Queries
	ReadQueries  *sqlc.Queries

	write *sql.DB
	read  *sql.DB
}

//go:embed migrations/*.sql
var migrationFS embed.FS

func New(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}

	if dir := filepath.Dir(cfg.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create database directory %q: %w", dir, err)
		}
	}

	dsn := buildDSN(cfg.Path)

	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open write connection: %w", err)
	}

	writeDB.SetMaxOpenConns(1)
	writeDB.SetConnMaxIdleTime(5 * time.Minute)

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writeDB.Close()
		return nil, fmt.Errorf("failed to open read connection: %w", err)
	}

	readDB.SetConnMaxIdleTime(5 * time.Minute)

	if err = runMigrations(ctx, writeDB); err != nil {
		_ = writeDB.Close()
		_ = readDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err = writeDB.PingContext(ctx); err != nil {
		_ = writeDB.Close()
		_ = readDB.Close()
		return nil, fmt.Errorf("failed to ping write connection: %w", err)
	}
	if err = readDB.PingContext(ctx); err != nil {
		_ = writeDB.Close()
		_ = readDB.Close()
		return nil, fmt.Errorf("failed to ping read connection: %w", err)
	}

	slog.InfoContext(ctx, "database connections established", "path", cfg.Path)

	return &Database{
		WriteQueries: sqlc.New(writeDB),
		ReadQueries:  sqlc.New(readDB),
		write:        writeDB,
		read:         readDB,
	}, nil
}

func buildDSN(path string) string {
	v := url.Values{}
	v.Add("_pragma", "journal_mode(WAL)")
	v.Add("_pragma", "busy_timeout(5000)")
	v.Add("_pragma", "synchronous(NORMAL)")
	v.Add("_pragma", "foreign_keys(ON)")

	return fmt.Sprintf("%s?%s", path, v.Encode())
}

func (db *Database) Close(ctx context.Context) error {
	var errs []error

	if err := db.write.Close(); err != nil {
		errs = append(errs, fmt.Errorf("write pool: %w", err))
	}
	if err := db.read.Close(); err != nil {
		errs = append(errs, fmt.Errorf("read pool: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database: %v", errs)
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
