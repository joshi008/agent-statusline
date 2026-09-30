package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joshi008/agent-statusline/internal/fsutil"
)

const (
	backupInfix = ".agent-statusline.bak."
	keepBackups = 5
)

// Backup copies path to <path>.agent-statusline.bak.<unixnano>. Missing file → "", nil. keep
// names the backups pruning must never delete (see pruneBackups); pass the function keepFunc
// builds for the harness's managed keys.
func Backup(path string, now time.Time, keep func(backups []string) map[string]bool) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	dst := fmt.Sprintf("%s%s%019d", path, backupInfix, now.UnixNano())
	if err := fsutil.WriteAtomic(dst, b, 0o600); err != nil {
		return "", err
	}
	pruneBackups(path, keep)
	return dst, nil
}

// pruneBackups keeps every backup keep names and prunes the rest (oldest first) down to
// keepBackups. keepFunc names, for each managed key, only the one backup uninstall would restore
// that key from (restoreSource: the newest backup where the key is not ours), so the user's
// original survives any number of reinstalls while the total stays at most
// keepBackups + number of managed keys (7 for Claude, 6 for Cursor and Codex).
func pruneBackups(path string, keep func(backups []string) map[string]bool) {
	bs, err := Backups(path)
	if err != nil {
		return
	}
	var protected map[string]bool
	if keep != nil {
		protected = keep(bs)
	}
	var prunable []string
	for _, p := range bs {
		if !protected[p] {
			prunable = append(prunable, p)
		}
	}
	need := len(prunable) - keepBackups
	for i := 0; i < len(prunable) && need > 0; i++ {
		if os.Remove(prunable[i]) == nil {
			need--
		}
	}
}

// Backups lists backups of path, oldest first.
func Backups(path string) ([]string, error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+backupInfix) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}
