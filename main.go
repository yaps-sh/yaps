package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

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
