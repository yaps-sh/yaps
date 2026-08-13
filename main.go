package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yaps-sh/yaps/internal/config"
	"github.com/yaps-sh/yaps/internal/database"
	"github.com/yaps-sh/yaps/internal/paste"
	"github.com/yaps-sh/yaps/internal/web"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.toml", "Path to TOML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				os.Stdout, &slog.HandlerOptions{
					Level: cfg.LogLevel(),
				},
			),
		),
	)

	slog.Debug(
		"Loaded config",
		"database.path", cfg.Database.Path,
		"http.base_url", cfg.HTTP.BaseURL,
		"paste.id_length", cfg.Paste.IDLength,
		"paste.defaults.anonymous.expiry_length", cfg.Paste.Defaults.Anonymous.ExpiryLength,
		"paste.defaults.anonymous.max_size", cfg.Paste.Defaults.Anonymous.MaxSize,
		"paste.defaults.authenticated.expiry_length", cfg.Paste.Defaults.Authenticated.ExpiryLength,
		"paste.defaults.authenticated.max_size", cfg.Paste.Defaults.Authenticated.MaxSize,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer closeWithTimeout("database", db.Close)

	pasteSvc := paste.New(db)
	go pasteSvc.StartReaper(ctx, 0)
	webHandler := web.NewHandler(pasteSvc, cfg)

	r := chi.NewRouter()

	if cfg.HTTP.ClientIPHeader != nil && *cfg.HTTP.ClientIPHeader != "" {
		r.Use(middleware.ClientIPFromHeader(*cfg.HTTP.ClientIPHeader))
	}

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	// TODO add ratelimiting middleware, make it configurable. should also only apply to the create paste endpoint

	r.Get("/", webHandler.Index)
	r.Get("/about", webHandler.About)
	r.Get("/static/chroma.css", webHandler.HighlightCSS)
	r.Get("/robots.txt", webHandler.RobotsTXT)

	r.Route(
		"/api/v1", func(r chi.Router) {
			r.Post("/paste", webHandler.CreatePaste)
		},
	)

	r.Get("/{id:[a-zA-Z0-9]+}", webHandler.GetPaste)
	r.Get("/{id:[a-zA-Z0-9]+}.{ext}", webHandler.GetPaste)

	srv := &http.Server{
		Addr:              ":3000",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
	}

	return nil
}

func closeWithTimeout(name string, close func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := close(ctx); err != nil {
		slog.Error("error during shutdown", slog.String("component", name), slog.Any("err", err))
	}
}
