package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestGet_FreshHitSkipsFetch(t *testing.T) {
	clk := &clock{time.Unix(1000, 0)}
	c := NewWithClock(t.TempDir(), "s1", clk.now)
	calls := 0
	fetch := func() (string, error) { calls++; return "v", nil }
	v, err := c.Get("git", 5*time.Second, fetch)
	require.NoError(t, err)
	require.Equal(t, "v", v)
	clk.t = clk.t.Add(4 * time.Second)
	v, _ = c.Get("git", 5*time.Second, fetch)
	require.Equal(t, "v", v)
	require.Equal(t, 1, calls)
}

func TestGet_ExpiredRefetches(t *testing.T) {
	clk := &clock{time.Unix(1000, 0)}
	c := NewWithClock(t.TempDir(), "s1", clk.now)
	n := 0
	fetch := func() (string, error) { n++; return string(rune('a' + n - 1)), nil }
	v, _ := c.Get("k", time.Second, fetch)
	require.Equal(t, "a", v)
	clk.t = clk.t.Add(2 * time.Second)
	v, _ = c.Get("k", time.Second, fetch)
	require.Equal(t, "b", v)
}

func TestCache_StaleOnError(t *testing.T) {
	clk := &clock{time.Unix(1000, 0)}
	c := NewWithClock(t.TempDir(), "s1", clk.now)
	_, _ = c.Get("pr", time.Second, func() (string, error) { return "old", nil })
	clk.t = clk.t.Add(time.Hour)
	v, err := c.Get("pr", time.Second, func() (string, error) { return "", errors.New("gh offline") })
	require.NoError(t, err)
	require.Equal(t, "old", v)
}

func TestGet_ErrorWithoutEntry(t *testing.T) {
	c := New(t.TempDir(), "s1")
	_, err := c.Get("pr", time.Second, func() (string, error) { return "", errors.New("boom") })
	require.EqualError(t, err, "boom")
}

func TestGet_SessionsAreIsolatedAndKeysSanitised(t *testing.T) {
	dir := t.TempDir()
	a, b := New(dir, "sess/../a"), New(dir, "b")
	_, _ = a.Get("cmd:k8s/x", time.Hour, func() (string, error) { return "A", nil })
	v, _ := b.Get("cmd:k8s/x", time.Hour, func() (string, error) { return "B", nil })
	require.Equal(t, "B", v)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		require.NotContains(t, e.Name(), "/")
	}
	require.Len(t, entries, 2)
}

func TestGet_UnwritableDirStillReturnsValue(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(f, nil, 0o644))
	c := New(filepath.Join(f, "sub"), "s")
	v, err := c.Get("k", time.Hour, func() (string, error) { return "v", nil })
	require.NoError(t, err)
	require.Equal(t, "v", v)
}

func TestDefaultDirAndPrune(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	require.Equal(t, "/x/agent-statusline", DefaultDir(env(map[string]string{"XDG_CACHE_HOME": "/x"}), "/h"))
	require.Equal(t, "/h/.cache/agent-statusline", DefaultDir(env(nil), "/h"))

	dir := t.TempDir()
	old, fresh := filepath.Join(dir, "old.json"), filepath.Join(dir, "fresh.json")
	require.NoError(t, os.WriteFile(old, nil, 0o644))
	require.NoError(t, os.WriteFile(fresh, nil, 0o644))
	now := time.Now()
	require.NoError(t, os.Chtimes(old, now.Add(-10*24*time.Hour), now.Add(-10*24*time.Hour)))
	n, err := Prune(dir, 7*24*time.Hour, now)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	_, err = os.Stat(fresh)
	require.NoError(t, err)
	n, err = Prune(filepath.Join(dir, "missing"), time.Hour, now)
	require.NoError(t, err)
	require.Zero(t, n)
}
