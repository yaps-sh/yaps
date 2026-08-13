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
	"sync"
	"time"

	"github.com/yaps-sh/yaps/internal/paste"
	"golang.org/x/sync/singleflight"
)

var errTombstoned = errors.New("ogimage: paste tombstoned")

type entryState struct {
	mu   sync.Mutex
	tomb bool
	refs int
}

type Cache struct {
	dir string
	sf  singleflight.Group
	smu sync.Mutex
	ids map[string]*entryState
}

func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("ogimage: create cache dir: %w", err)
	}
	return &Cache{dir: dir, ids: make(map[string]*entryState)}, nil
}

func (c *Cache) path(id string) string {
	return filepath.Join(c.dir, id+".png")
}

func (c *Cache) acquire(id string) *entryState {
	c.smu.Lock()
	defer c.smu.Unlock()
	s, ok := c.ids[id]
	if !ok {
		s = &entryState{}
		c.ids[id] = s
	}
	s.refs++
	return s
}

func (c *Cache) release(s *entryState, id string) {
	c.smu.Lock()
	defer c.smu.Unlock()
	s.refs--
	if s.refs == 0 {
		delete(c.ids, id)
	}
}

func (c *Cache) Delete(id string) error {
	s := c.acquire(id)
	s.mu.Lock()
	s.tomb = true
	err := os.Remove(c.path(id))
	s.mu.Unlock()
	c.release(s, id)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("ogimage: delete cache %q: %w", id, err)
	}
	return nil
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
			s := c.acquire(entry.ID)
			released := false
			release := func() {
				if !released {
					c.release(s, entry.ID)
					released = true
				}
			}
			defer release()

			b, err := Generate(entry)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.tomb {
				slog.Debug("ogimage: paste tombstoned mid-flight, skipping cache write", "id", entry.ID)
				return nil, errTombstoned
			}
			if werr := writeAtomic(p, b); werr != nil {
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

func writeAtomic(p string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(p), ".og-*.png.tmp")
	if err != nil {
		return fmt.Errorf("tempfile: %w", err)
	}
	tmpName := tmp.Name()
	if _, werr := tmp.Write(b); werr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write: %w", werr)
	}
	if cerr := tmp.Close(); cerr != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close: %w", cerr)
	}
	if rerr := os.Rename(tmpName, p); rerr != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename: %w", rerr)
	}
	return nil
}

func (c *Cache) Serve(w http.ResponseWriter, r *http.Request, entry *paste.Entry) {
	if entry.BurnAfterRead {
		http.NotFound(w, r)
		return
	}
	b, err := c.Get(entry)
	if err != nil {
		if errors.Is(err, errTombstoned) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(r.Context(), "ogimage: generate failed", "id", entry.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	etag := etagFor(b)
	w.Header().Set("Content-Type", "image/png")
	ttl := time.Until(entry.ExpiresAt)
	switch {
	case ttl <= 0:
		w.Header().Set("Cache-Control", "no-store")
	case ttl.Seconds() <= 86400:
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, must-revalidate", int(ttl.Seconds())))
	default:
		w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
	}
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
