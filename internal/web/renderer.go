package web

import (
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Index().Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "failed to render index", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (rn *Renderer) RenderHighlightCSS(w http.ResponseWriter, r *http.Request) {
	css, err := highlightCSS()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to render highlight css", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(css))
}

func (rn *Renderer) RenderView(w http.ResponseWriter, r *http.Request, entry *paste.Entry, ext string) {
	highlighted, err := highlight(entry.DetectedLanguage, ext, entry.Content)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to highlight", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.View(entry, highlighted).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "failed to render view", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
