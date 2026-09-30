package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/joshi008/agent-statusline/internal/codex"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/fsutil"
)

type claudeStatusLine struct {
	Type            string `json:"type"`
	Command         string `json:"command"`
	RefreshInterval int    `json:"refreshInterval,omitempty"`
}

type commandHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type cursorStatusLine struct {
	Type             string `json:"type"`
	Command          string `json:"command"`
	Padding          int    `json:"padding"`
	UpdateIntervalMs int    `json:"updateIntervalMs,omitempty"`
	TimeoutMs        int    `json:"timeoutMs,omitempty"`
}

func InstallClaude(path, bin string, cfg *config.Config, now time.Time) (Result, error) {
	res := Result{Harness: "claude", Path: path}
	src, err := readJSONObject(path)
	if err != nil {
		return res, err
	}
	if cur := gjson.GetBytes(src, "statusLine.command"); cur.Exists() && !strings.Contains(cur.String(), Marker) {
		res.Warnings = append(res.Warnings, "replaced your existing statusLine ("+cur.String()+"); `uninstall claude` restores it")
	}
	if gjson.GetBytes(src, "disableAllHooks").Bool() {
		res.Warnings = append(res.Warnings, "disableAllHooks is true, so Claude Code will not run any status line")
	}
	q := shellQuote(bin)
	out, err := setKey(src, "statusLine", claudeStatusLine{Type: "command", Command: q + " render --harness claude",
		RefreshInterval: cfg.Claude.RefreshInterval})
	if err != nil {
		return res, err
	}
	if cfg.Claude.Subagents.Enabled {
		if cur := gjson.GetBytes(src, "subagentStatusLine.command"); cur.Exists() && !strings.Contains(cur.String(), Marker) {
			res.Warnings = append(res.Warnings, "replaced your existing subagentStatusLine ("+cur.String()+"); `uninstall claude` restores it")
		}
		if out, err = setKey(out, "subagentStatusLine", commandHook{Type: "command", Command: q + " render-subagents"}); err != nil {
			return res, err
		}
	} else if isOurs(out, "subagentStatusLine") {
		// Subagent rows are being turned off: hand back the user's own subagentStatusLine (the
		// same value uninstall would restore), or drop ours when they never had one.
		bs, _ := Backups(path)
		if raw, backupPath, found := findRestoreValue(bs, "claude", "subagentStatusLine"); found {
			if out, err = setKeyRaw(out, "subagentStatusLine", raw); err != nil {
				return res, err
			}
			res.Warnings = append(res.Warnings, "restored your previous subagentStatusLine from "+backupPath)
		} else {
			out = deleteIfOurs(out, "subagentStatusLine")
		}
	}
	return commitJSON(res, path, src, out, now, keepFunc("claude", "statusLine", "subagentStatusLine"))
}

func InstallCursor(path, bin string, cfg *config.Config, now time.Time) (Result, error) {
	res := Result{Harness: "cursor", Path: path}
	if !filepath.IsAbs(bin) {
		return res, errors.New("cursor runs the command without a shell, so the binary path must be absolute")
	}
	if strings.ContainsAny(bin, " \t") {
		return res, errors.New("cursor splits the command on spaces; move agent-statusline to a path without a space")
	}
	src, err := readJSONObject(path)
	if err != nil {
		return res, err
	}
	if cur := gjson.GetBytes(src, "statusLine.command"); cur.Exists() && !strings.Contains(cur.String(), Marker) {
		res.Warnings = append(res.Warnings, "replaced your existing statusLine ("+cur.String()+"); `uninstall cursor` restores it")
	}
	out, err := setKey(src, "statusLine", cursorStatusLine{Type: "command", Command: bin + " render --harness cursor",
		Padding: 0, UpdateIntervalMs: cfg.Cursor.UpdateIntervalMs, TimeoutMs: cfg.Cursor.TimeoutMs})
	if err != nil {
		return res, err
	}
	return commitJSON(res, path, src, out, now, keepFunc("cursor", "statusLine"))
}

func InstallCodex(path string, cfg *config.Config, now time.Time) (Result, error) {
	res := Result{Harness: "codex", Path: path}
	ids, dropped := codex.Translate(cfg.Codex.Items)
	res.Dropped = dropped
	if len(ids) == 0 {
		return res, errors.New("codex.items has no segment that Codex can show")
	}
	src, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}
	if cur, ok, _ := codex.GetStatusLine(src); ok && !keyOwned("codex", src, "status_line") {
		res.Warnings = append(res.Warnings, "replaced your existing status_line ("+codex.FormatArray(cur)+"); `uninstall codex` restores it")
	}
	out, err := codex.SetStatusLineWithComment(src, ids, codex.ManagedComment)
	if err != nil {
		return res, err
	}
	return commit(res, path, src, out, now, keepFunc("codex", "status_line"))
}

func commitJSON(res Result, path string, src, out []byte, now time.Time, keep func(backups []string) map[string]bool) (Result, error) {
	if !json.Valid(out) {
		return res, errors.New("internal error: edit produced invalid JSON; file left untouched")
	}
	return commit(res, path, src, out, now, keep)
}

// commit backs up and writes only when the content changed (or the file does not exist yet).
// keep is threaded through to Backup so pruning knows which existing backups to protect.
func commit(res Result, path string, src, out []byte, now time.Time, keep func(backups []string) map[string]bool) (Result, error) {
	if _, err := os.Stat(path); err == nil && bytes.Equal(src, out) {
		return res, nil
	}
	b, err := Backup(path, now, keep)
	if err != nil {
		return res, err
	}
	res.Backup = b
	if err := fsutil.WriteAtomic(path, out, 0o644); err != nil {
		return res, err
	}
	res.Changed = true
	return res, nil
}
