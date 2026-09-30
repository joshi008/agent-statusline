package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/joshi008/agent-statusline/internal/codex"
)

// Uninstall removes our entries. If the file's current status line isn't ours, nothing is
// touched (Changed stays false): there is nothing of ours to remove, and we must never disturb
// a status line someone else set up since we last ran. Otherwise our keys are stripped and, for
// each key we actually remove, if an earlier backup (skipping any backup where that specific key
// is itself ours) had a value of its own for it, only that value is restored — never the rest of
// the file, and never a value some other key's ownership happened to shadow.
func Uninstall(harness, path string, now time.Time) (Result, error) {
	res := Result{Harness: harness, Path: path}
	switch harness {
	case "claude", "cursor", "codex":
	default:
		return res, fmt.Errorf("unknown harness %q", harness)
	}
	src, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return res, nil
	}
	if err != nil {
		return res, err
	}

	switch harness {
	case "claude":
		return uninstallJSON(res, src, path, now, "statusLine", "subagentStatusLine")
	case "cursor":
		return uninstallJSON(res, src, path, now, "statusLine")
	default:
		return uninstallCodex(res, src, path, now)
	}
}

// uninstallJSON handles Claude and Cursor: both keep their status line under one or more
// top-level JSON keys, each recognised as ours via isOurs on "<key>.command". Only a key that
// is currently ours in src is ever touched — one the user has already replaced or deleted since
// we last ran is left exactly as they left it. Each key's replacement is chosen independently:
// the newest backup where THAT key specifically isn't ours, never a backup skipped wholesale
// just because some other key in it happens to be ours.
func uninstallJSON(res Result, src []byte, path string, now time.Time, keys ...string) (Result, error) {
	owns := func(b []byte) bool {
		for _, k := range keys {
			if isOurs(b, k) {
				return true
			}
		}
		return false
	}
	if !owns(src) {
		return res, nil // the current status line isn't ours: nothing to remove
	}

	bs, _ := Backups(path) // oldest first; a listing error just means no candidates to restore from

	out := src
	for _, k := range keys {
		if !isOurs(src, k) {
			continue // not currently ours: the user replaced or removed it; leave it exactly as-is
		}
		raw, backupPath, found := findRestoreValue(bs, res.Harness, k)
		if found {
			// The key is still there (it's ours) — swap its value in place so it keeps its
			// original position, rather than deleting it and re-appending the restored value.
			if merged, err := sjson.SetRawBytes(out, k, raw); err == nil {
				out = merged
				res.Warnings = append(res.Warnings, "restored your previous "+k+" from "+backupPath)
			}
			continue
		}
		if stripped, err := sjson.DeleteBytes(out, k); err == nil {
			out = stripped
		}
	}
	return commitJSON(res, path, src, out, now, keepFunc(res.Harness, keys...))
}

// findRestoreValue returns the raw JSON value key had in the backup restoreSource picks for it
// (the newest one where key is not ours) — the value uninstall should restore key to. found is
// false either when that backup never had key (key should simply stay deleted) or when every
// backup is ours for key.
func findRestoreValue(backups []string, harness, key string) (raw []byte, backupPath string, found bool) {
	p, prev, ok := restoreSource(backups, harness, key)
	if !ok {
		return nil, "", false
	}
	if val := gjson.GetBytes(prev, key); val.Exists() {
		return []byte(val.Raw), p, true
	}
	return nil, "", false // the chosen backup has no value for key: leave it deleted
}

// uninstallCodex handles Codex: ownership is recognised via the ManagedComment marker
// SetStatusLineWithComment writes on the status_line line.
func uninstallCodex(res Result, src []byte, path string, now time.Time) (Result, error) {
	if !keyOwned("codex", src, "status_line") {
		return res, nil // the current status_line isn't ours: nothing to remove
	}
	// With a previous value of the user's own, overwrite our line with it in place (without the
	// marker), so a dotted top-level tui.status_line stays where it was. Only when there is
	// nothing to restore is our line removed.
	if bs, err := Backups(path); err == nil {
		if p, prev, ok := restoreSource(bs, "codex", "status_line"); ok {
			if ids, ok, _ := codex.GetStatusLine(prev); ok {
				if out, err := codex.SetStatusLine(src, ids); err == nil {
					res.Warnings = append(res.Warnings, "restored your previous status_line from "+p)
					return commit(res, path, src, out, now, keepFunc("codex", "status_line"))
				}
			}
		}
	}
	out, _, err := codex.RemoveStatusLine(src)
	if err != nil {
		return res, err
	}
	return commit(res, path, src, out, now, keepFunc("codex", "status_line"))
}
