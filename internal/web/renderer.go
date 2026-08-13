package web

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/yaps-sh/yaps/internal/build"
	"github.com/yaps-sh/yaps/internal/httpheaders"
	"github.com/yaps-sh/yaps/internal/paste"
	"github.com/yaps-sh/yaps/internal/web/templates"
)

func writeTempl(w http.ResponseWriter, r *http.Request, comp templ.Component, label string) {
	var buf bytes.Buffer
	if err := comp.Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "failed to render "+label, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(buf.Bytes())
}

func renderIndex(w http.ResponseWriter, r *http.Request, baseURL string) {
	writeTempl(w, r, templates.Index(baseURL), "index")
}

func renderAbout(w http.ResponseWriter, r *http.Request, bld build.Info, latest string, baseURL string) {
	writeTempl(w, r, templates.About(bld, latest, baseURL), "about")
}

func renderHighlightCSS(w http.ResponseWriter, r *http.Request) {
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
	if httpheaders.ETagMatches(r.Header.Values("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write([]byte(css))
}

func renderView(w http.ResponseWriter, r *http.Request, entry *paste.Entry, ext string, baseURL string) {
	highlighted, err := highlight(entry.DetectedLanguage, ext, entry.Content)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to highlight", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeTempl(w, r, templates.View(entry, highlighted, baseURL), "view")
}
