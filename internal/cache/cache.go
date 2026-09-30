// Package cache stores slow lookups (git, gh, custom commands) per session in small JSON files,
// so a render that runs every few hundred milliseconds rarely spawns a process.
package cache

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joshi008/agent-statusline/internal/fsutil"
)

type Cache struct {
	dir     string
	session string
	now     func() time.Time
}

type entry struct {
	Value     string `json:"value"`
	WrittenAt int64  `json:"written_at"` // unix milliseconds
}

func New(dir, session string) *Cache { return NewWithClock(dir, session, time.Now) }

func NewWithClock(dir, session string, now func() time.Time) *Cache {
	if session == "" {
		session = "nosession"
	}
	return &Cache{dir: dir, session: sanitize(session), now: now}
}

// DefaultDir is $XDG_CACHE_HOME/agent-statusline or ~/.cache/agent-statusline.
func DefaultDir(getenv func(string) string, home string) string {
	if x := getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "agent-statusline")
	}
	return filepath.Join(home, ".cache", "agent-statusline")
}

var unsafeRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func sanitize(s string) string {
	s = strings.ReplaceAll(unsafeRe.ReplaceAllString(s, "_"), "..", "_")
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func (c *Cache) file(key string) string {
	return filepath.Join(c.dir, c.session+"__"+sanitize(key)+".json")
}

func (c *Cache) read(key string) (entry, bool) {
	b, err := os.ReadFile(c.file(key))
	if err != nil {
		return entry{}, false
	}
	var e entry
	if json.Unmarshal(b, &e) != nil {
		return entry{}, false
	}
	return e, true
}

// Get returns a fresh cached value, or calls fetch. On fetch failure a stale value wins.
func (c *Cache) Get(key string, ttl time.Duration, fetch func() (string, error)) (string, error) {
	e, ok := c.read(key)
	if ok && c.now().Sub(time.UnixMilli(e.WrittenAt)) < ttl {
		return e.Value, nil
	}
	v, err := fetch()
	if err != nil {
		if ok {
			return e.Value, nil
		}
		return "", err
	}
	if b, err := json.Marshal(entry{Value: v, WrittenAt: c.now().UnixMilli()}); err == nil {
		_ = fsutil.WriteAtomic(c.file(key), b, 0o644)
	}
	return v, nil
}

// Prune deletes cache files older than maxAge. A missing dir is not an error.
func Prune(dir string, maxAge time.Duration, now time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, de := range entries {
		info, err := de.Info()
		if err != nil || de.IsDir() {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			if os.Remove(filepath.Join(dir, de.Name())) == nil {
				n++
			}
		}
	}
	return n, nil
}
