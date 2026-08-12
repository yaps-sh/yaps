package web

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/yaps-sh/yaps/internal/paste"
	"github.com/yaps-sh/yaps/internal/web/templates"
)

type Renderer struct{}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (rn *Renderer) RenderIndex(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := templates.Index().Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "failed to render index", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(buf.Bytes())
}

func (rn *Renderer) RenderHighlightCSS(w http.ResponseWriter, r *http.Request) {
	css, err := highlightCSSCache()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to render highlight css", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	etag := highlightCSSETag(css)
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write([]byte(css))
}

func (rn *Renderer) RenderView(w http.ResponseWriter, r *http.Request, entry *paste.Entry, ext string) {
	highlighted, err := highlight(entry.DetectedLanguage, ext, entry.Content)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to highlight", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := templates.View(entry, highlighted, entry.Content).Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "failed to render view", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(buf.Bytes())
}
