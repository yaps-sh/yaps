package ogimage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaps-sh/yaps/internal/paste"
	"golang.org/x/sync/singleflight"
)

type Cache struct {
	dir string
	sf  singleflight.Group
}

func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("ogimage: create cache dir: %w", err)
	}
	return &Cache{dir: dir}, nil
}

func (c *Cache) path(id string) string {
	return filepath.Join(c.dir, id+".png")
}

func (c *Cache) Delete(id string) {
	_ = os.Remove(c.path(id))
}

func (c *Cache) Get(entry *paste.Entry) ([]byte, error) {
	p := c.path(entry.ID)
	if b, err := os.ReadFile(p); err == nil {
		slog.Debug("ogimage: cache hit", "id", entry.ID)
		return b, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("ogimage: read cache: %w", err)
	}

	v, err, _ := c.sf.Do(
		entry.ID, func() (any, error) {
			if b, err := os.ReadFile(p); err == nil {
				slog.Debug("ogimage: cache hit (post-flight)", "id", entry.ID)
				return b, nil
			}
			slog.Debug("ogimage: generating", "id", entry.ID)
			b, err := Generate(entry)
			if err != nil {
				return nil, err
			}
			if werr := os.WriteFile(p, b, 0o640); werr != nil {
				slog.Warn("ogimage: cache write failed", "id", entry.ID, "err", werr)
			}
			return b, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (c *Cache) Serve(w http.ResponseWriter, r *http.Request, entry *paste.Entry) {
	if entry.BurnAfterRead {
		http.NotFound(w, r)
		return
	}
	b, err := c.Get(entry)
	if err != nil {
		slog.ErrorContext(r.Context(), "ogimage: generate failed", "id", entry.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	etag := etagFor(b)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", etag)
	if etagMatches(r.Header.Values("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(b)
}

func etagFor(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("\"%x\"", hex.EncodeToString(sum[:8]))
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
