package web

import (
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
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (rn *Renderer) RenderView(w http.ResponseWriter, r *http.Request, entry *paste.Entry) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.View(entry).Render(r.Context(), w); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
