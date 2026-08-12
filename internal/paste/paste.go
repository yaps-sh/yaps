package paste

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/alecthomas/chroma/v3/lexers"
	"github.com/yaps-sh/yaps/internal/database"
	"github.com/yaps-sh/yaps/internal/database/sqlc"
)

const (
	base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type Paste struct {
	db *database.Database
}

type Entry struct {
	ID               string
	Filename         *string
	DetectedLanguage string
	Content          string
	SizeBytes        int64
	ViewCount        int64
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

type CreateParams struct {
	Content   string
	Filename  *string
	Language  *string
	ExpiresIn time.Duration
	IDLength  int
}

func New(db *database.Database) *Paste {
	return &Paste{
		db: db,
	}
}

func (p *Paste) Create(ctx context.Context, params CreateParams) (*Entry, error) {
	id, err := generateID(params.IDLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	language := params.Language
	var resolvedLanguage string
	if language != nil && *language != "" {
		resolvedLanguage = *language
	} else {
		resolvedLanguage = detectLanguage(params.Filename, params.Content)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(params.ExpiresIn)
	sizeBytes := int64(len(params.Content))

	filename := params.Filename

	err = p.db.WriteQueries.CreatePaste(
		ctx, sqlc.CreatePasteParams{
			ID:               id,
			Filename:         filename,
			DetectedLanguage: resolvedLanguage,
			Content:          params.Content,
			SizeBytes:        sizeBytes,
			ExpiresAt:        expiresAt.Format(time.RFC3339),
			CreatedAt:        now.Format(time.RFC3339),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create paste: %w", err)
	}

	return &Entry{
		ID:               id,
		Filename:         filename,
		DetectedLanguage: resolvedLanguage,
		Content:          params.Content,
		SizeBytes:        sizeBytes,
		ExpiresAt:        expiresAt,
		ViewCount:        0,
		CreatedAt:        now,
	}, nil
}

func (p *Paste) Get(ctx context.Context, id string) (*Entry, error) {
	row, err := p.db.ReadQueries.GetPaste(
		ctx, sqlc.GetPasteParams{
			ID:  id,
			Now: time.Now().UTC().Format(time.RFC3339),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get paste: %w", err)
	}

	e, err := fromRow(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to entry: %w", err)
	}

	return e, nil
}

func (p *Paste) IncrementViewCount(id string) {
	go func() {
		incCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if incErr := p.incrementViewCount(incCtx, id); incErr != nil {
			slog.WarnContext(incCtx, "failed to increment view count", "id", id, "err", incErr)
		}
	}()
}

func (p *Paste) incrementViewCount(ctx context.Context, id string) error {
	if err := p.db.WriteQueries.IncrementViewCount(ctx, id); err != nil {
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	return nil
}

func fromRow(row sqlc.Paste) (*Entry, error) {
	expiresAt, err := time.Parse(time.RFC3339, row.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires at: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created at: %w", err)
	}

	filename := row.Filename

	return &Entry{
		ID:               row.ID,
		Filename:         filename,
		DetectedLanguage: row.DetectedLanguage,
		Content:          row.Content,
		SizeBytes:        row.SizeBytes,
		ViewCount:        row.ViewCount,
		ExpiresAt:        expiresAt,
		CreatedAt:        createdAt,
	}, nil
}

func generateID(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("id length must be greater than zero, got %d", length)
	}

	b := make([]byte, length)
	alphabetLength := byte(len(base62Alphabet))
	maxValid := byte(256 - (256 % int(alphabetLength)))
	buf := make([]byte, 1)

	for i := range length {
		for {
			if _, err := rand.Read(buf); err != nil {
				return "", fmt.Errorf("failed to read random bytes: %w", err)
			}

			if buf[0] < maxValid {
				b[i] = base62Alphabet[buf[0]%alphabetLength]
				break
			}
		}
	}

	return string(b), nil
}

func detectLanguage(filename *string, content string) string {
	if filename != nil && *filename != "" {
		if lexer := lexers.Match(*filename); lexer != nil {
			slog.Debug("detected language from filename", "language", lexer.Config().Name)
			return lexer.Config().Name
		}
	}

	if lexer := lexers.Analyse(content); lexer != nil {
		slog.Debug("detected language", "language", lexer.Config().Name)
		return lexer.Config().Name
	}

	slog.Debug("falling back to plaintext")

	return "plaintext"
}
