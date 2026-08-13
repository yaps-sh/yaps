package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"

	"github.com/yaps-sh/yaps/internal/build"
	"github.com/yaps-sh/yaps/internal/paste"
	"github.com/yaps-sh/yaps/internal/web/templates"
)

func renderIndex(w http.ResponseWriter, r *http.Request) {
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

func renderAbout(w http.ResponseWriter, r *http.Request, bld build.Info, latest string) {
	var buf bytes.Buffer
	if err := templates.About(bld, latest).Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "failed to render about", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(buf.Bytes())
}

func etagMatches(values []string, etag string) bool {
	want := strings.Trim(etag, "\"")
	for _, v := range values {
		for _, tag := range strings.Split(v, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if tag == "*" {
				return true
			}
			t := strings.TrimPrefix(tag, "W/")
			t = strings.TrimPrefix(t, "w/")
			t = strings.Trim(t, "\"")
			if t == want {
				return true
			}
		}
	}
	return false
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
	if etagMatches(r.Header.Values("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write([]byte(css))
}

func renderView(w http.ResponseWriter, r *http.Request, entry *paste.Entry, ext string) {
	highlighted, err := highlight(entry.DetectedLanguage, ext, entry.Content)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to highlight", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := templates.View(entry, highlighted).Render(r.Context(), &buf); err != nil {
		slog.ErrorContext(r.Context(), "failed to render view", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(buf.Bytes())
}
