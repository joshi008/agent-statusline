package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteAtomic_CreatesDirsAndFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "b", "f.json")
	require.NoError(t, WriteAtomic(p, []byte("x"), 0o600))
	b, _ := os.ReadFile(p)
	require.Equal(t, "x", string(b))
	fi, _ := os.Stat(p)
	require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestWriteAtomic_KeepsModeAndSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.json")
	require.NoError(t, os.WriteFile(real, []byte("old"), 0o640))
	link := filepath.Join(dir, "link.json")
	require.NoError(t, os.Symlink(real, link))

	require.NoError(t, WriteAtomic(link, []byte("new"), 0o644))

	fi, err := os.Lstat(link)
	require.NoError(t, err)
	require.True(t, fi.Mode()&os.ModeSymlink != 0, "symlink must survive")
	b, _ := os.ReadFile(real)
	require.Equal(t, "new", string(b))
	rfi, _ := os.Stat(real)
	require.Equal(t, os.FileMode(0o640), rfi.Mode().Perm())
	left, _ := filepath.Glob(filepath.Join(dir, ".*tmp*"))
	require.Empty(t, left, "no temp files left behind")
}
