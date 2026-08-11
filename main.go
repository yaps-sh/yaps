package main

import (
	"context"
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	// TODO: configurable?
	// r.Use(middleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)

	r.Get(
		"/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		},
	)

	r.Route(
		"/api/v1", func(r chi.Router) {
			r.Post(
				"/paste", func(w http.ResponseWriter, r *http.Request) {
					//
				},
			)

			r.Get(
				"/paste/{id}", func(w http.ResponseWriter, r *http.Request) {
					//
				},
			)
		},
	)

	// TODO add regex to only allow the limited id values (letters, numbers)
	r.Get("/{id}", viewPaste)
	r.Get("/{id}.{ext}", viewPaste)

	srv := &http.Server{
		Addr:         ":3000", // TODO: config this too
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func viewPaste(w http.ResponseWriter, r *http.Request) {
	// TODO all this.
	// id := chi.URLParam(r, "id")
	// extension := chi.URLParam(r, "ext")
	// viewMode := r.URL.Query().Get("view")

}

func closeWithTimeout(name string, close func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := close(ctx); err != nil {
		slog.Error("error during shutdown", slog.String("component", name), slog.Any("err", err))
	}
}
