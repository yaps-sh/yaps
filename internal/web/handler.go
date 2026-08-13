package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/yaps-sh/yaps/internal/build"
	"github.com/yaps-sh/yaps/internal/config"
	"github.com/yaps-sh/yaps/internal/paste"
)

type Handler struct {
	pasteSvc      *paste.Paste
	validatorSvc  *validator.Validate
	cfg           *config.Config
	buildInfo     build.Info
	latestVersion string
}

type CreatePasteRequest struct {
	Content   string  `json:"content" validate:"required,utf8,max_size"`
	Filename  *string `json:"filename" validate:"omitempty"`
	Language  *string `json:"language" validate:"omitempty"`
	ExpiresIn *int64  `json:"expires_in" validate:"omitempty,gt=0"`
}

type CreatePasteResponse struct {
	ID               string `json:"id"`
	URL              string `json:"url"`
	Language         string `json:"language"`
	LanguageDetected bool   `json:"language_detected"`
	SizeBytes        int64  `json:"size_bytes"`
	ExpiresAt        string `json:"expires_at"`
	CreatedAt        string `json:"created_at"`
}

type ErrorResponse struct {
	Error      string `json:"error"`
	LimitBytes int64  `json:"limit_bytes,omitempty"`
	RetryAfter int64  `json:"retry_after,omitempty"`
}

func NewHandler(pasteSvc *paste.Paste, cfg *config.Config, bld build.Info, latestVersion string) *Handler {
	return &Handler{
		pasteSvc:      pasteSvc,
		validatorSvc:  newValidator(int64(cfg.Paste.Defaults.Anonymous.MaxSize)),
		cfg:           cfg,
		buildInfo:     bld,
		latestVersion: latestVersion,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	renderIndex(w, r)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	renderAbout(w, r, h.buildInfo, h.latestVersion)
}

func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":      h.buildInfo.Version,
		"commit":       h.buildInfo.Commit,
		"date":         h.buildInfo.Date,
		"latest_known": h.latestVersion,
	})
}

func (h *Handler) HighlightCSS(w http.ResponseWriter, r *http.Request) {
	renderHighlightCSS(w, r)
}

func (h *Handler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	// TODO switch this based on the auth status
	limit := int64(h.cfg.Paste.Defaults.Anonymous.MaxSize)
	r.Body = http.MaxBytesReader(w, r.Body, limit+1)

	var req CreatePasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(
				w, http.StatusRequestEntityTooLarge, ErrorResponse{
					Error:      "paste exceeds size limit for upload",
					LimitBytes: limit,
				},
			)

			return
		}

		writeJSON(
			w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid request body",
			},
		)

		return
	}

	if err := h.validatorSvc.Struct(req); err != nil {
		msg := "invalid request payload"
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) && len(vErrs) > 0 {
			fe := vErrs[0]
			if fe.Field() == "Content" && fe.Tag() == "max_size" {
				writeJSON(
					w, http.StatusRequestEntityTooLarge, ErrorResponse{
						Error:      "paste exceeds size limit for upload",
						LimitBytes: limit,
					},
				)
				return
			}
			switch {
			case fe.Field() == "Content" && fe.Tag() == "required":
				msg = "content is required"
			case fe.Field() == "Content" && fe.Tag() == "utf8":
				msg = "content must be valid UTF-8 text"
			case fe.Field() == "ExpiresIn" && fe.Tag() == "gt":
				msg = "expires_in must be greater than 0"
			}
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: msg})
		return
	}

	expiresIn := time.Duration(h.cfg.Paste.Defaults.Anonymous.ExpiryLength)
	if req.ExpiresIn != nil {
		requested := time.Duration(*req.ExpiresIn) * time.Second
		if requested > 0 && requested < expiresIn {
			expiresIn = requested
		}
	}

	entry, err := h.pasteSvc.Create(
		r.Context(), paste.CreateParams{
			Content:   req.Content,
			Filename:  req.Filename,
			Language:  req.Language,
			ExpiresIn: expiresIn,
			IDLength:  h.cfg.Paste.IDLength,
		},
	)

	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create paste", "error", err)
		writeJSON(
			w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to create paste",
			},
		)

		return
	}

	languageProvided := req.Language != nil && *req.Language != ""

	writeJSON(
		w, http.StatusCreated, CreatePasteResponse{
			ID:               entry.ID,
			URL:              fmt.Sprintf("%s/%s", h.cfg.HTTP.BaseURL, entry.ID),
			Language:         entry.DetectedLanguage,
			LanguageDetected: !languageProvided,
			SizeBytes:        entry.SizeBytes,
			ExpiresAt:        entry.ExpiresAt.Format(time.RFC3339),
			CreatedAt:        entry.CreatedAt.Format(time.RFC3339),
		},
	)
}

func (h *Handler) GetPaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry, err := h.pasteSvc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch paste", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Robots-Tag", "noindex, nofollow")

	extension := chi.URLParam(r, "ext")
	viewMode := r.URL.Query().Get("view")

	switch viewMode {
	case "raw":
		writeRaw(w, entry.Content)
		return

	case "preview":
		// TODO: this needs to be the preview system
		h.pasteSvc.IncrementViewCount(id)
		renderView(w, r, entry, extension)
		return

	case "":
		h.pasteSvc.IncrementViewCount(id)
		renderView(w, r, entry, extension)
		return

	default:
		http.Redirect(w, r, "/"+id, http.StatusFound)
		return
	}

}

func writeRaw(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) RobotsTXT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte("User-agent: *\nAllow: /$\nDisallow: /\n"))
}
