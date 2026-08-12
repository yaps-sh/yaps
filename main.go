package main

import (
	"context"
	"errors"
	"flag"
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

const shutdownTimeout = 10 * time.Second

func main() {
	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				os.Stdout, &slog.HandlerOptions{
					Level: slog.LevelDebug, // TODO: change this to level and have it change on config load?
				},
			),
		),
	)

	configPath := flag.String("config", "config.toml", "Path to TOML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Debug("Loaded config", "config", cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		slog.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer closeWithTimeout("database", db.Close)

	pasteSvc := paste.New(db)
	rendererSvc := web.NewRenderer()
	webHandler := web.NewHandler(pasteSvc, rendererSvc, cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	// TODO: configurable?
	// ClientIPFromRemoteAddr for default option
	// r.Use(middleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	// TODO add ratelimiting middleware, make it configurable. should also only apply to the create paste endpoint

	r.Get("/", webHandler.Index)
	r.Get("/static/chroma.css", webHandler.HighlightCSS)

	r.Route(
		"/api/v1", func(r chi.Router) {
			r.Post("/paste", webHandler.CreatePaste)
		},
	)

	r.Get("/{id:[a-zA-Z0-9]+}", webHandler.GetPaste)
	r.Get("/{id:[a-zA-Z0-9]+}.{ext}", webHandler.GetPaste)

	srv := &http.Server{
		Addr:         ":3000", // TODO: config this too
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
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
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during server shutdown", "error", err)
		}
	}
}

func closeWithTimeout(name string, close func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := close(ctx); err != nil {
		slog.Error("error during shutdown", slog.String("component", name), slog.Any("err", err))
	}
}
