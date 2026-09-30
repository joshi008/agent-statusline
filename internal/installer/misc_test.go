package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func writeFileForTest(p, s string) error { return os.WriteFile(p, []byte(s), 0o644) }

func TestBackup_KeepsFive(t *testing.T) {
	p := write(t, "settings.json", "{}")
	never := func([]string) map[string]bool { return nil } // nothing protected: plain oldest-first pruning
	for i := 0; i < 7; i++ {
		_, err := Backup(p, t0.Add(time.Duration(i)*time.Second), never)
		require.NoError(t, err)
	}
	bs, err := Backups(p)
	require.NoError(t, err)
	require.Len(t, bs, 5)
	b, err := Backup(filepath.Join(t.TempDir(), "missing.json"), t0, never)
	require.NoError(t, err)
	require.Empty(t, b)
}

// Fix round 2, controller ruling: reinstalling repeatedly must never prune away the one backup
// that holds the user's true original settings, even once more than keepBackups reinstalls have
// happened — otherwise uninstall would have nothing left to restore from.
func TestBackup_ReinstallsNeverPruneTheOriginal(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	cfg := config.Default()
	for i := 0; i < 7; i++ {
		cfg.Claude.RefreshInterval = 60 + i // force a real change each cycle, so each one backs up
		_, err := InstallClaude(p, bin, cfg, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}
	bs, err := Backups(p)
	require.NoError(t, err)
	// Fix round 3: pruning now protects the one backup that isn't ours (the original) in
	// addition to keeping keepBackups of the rest, so the total can sit above keepBackups —
	// here 5 (pruned, all-ours intermediate states) + 1 (protected original) = 6.
	require.Len(t, bs, keepBackups+1, "keepBackups regular backups plus the one protected original")

	res, err := Uninstall("claude", p, t0.Add(10*time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, "bash /Users/me/.claude/statusline-command.sh", gjson.Get(read(t, p), "statusLine.command").String(),
		"the original (pre-install) statusLine must still be recoverable after 7 reinstalls")
}

// Fix round 3, IMPORTANT: pruning must decide "is this backup ours" the same way uninstall does
// (per managed key), not by searching the whole backup for the marker string anywhere in it —
// otherwise an unrelated mention of "agent-statusline" (e.g. in a permissions allow-list) makes
// pruning think the user's real original backup is one of ours, and it gets pruned away.
func TestBackup_ProtectsOriginalEvenWhenMarkerAppearsElsewhereInFile(t *testing.T) {
	original := `{"permissions": {"allow": ["Bash(agent-statusline doctor)"]}, "statusLine": {"command": "bash mine.sh"}}` + "\n"
	p := write(t, "settings.json", original)
	cfg := config.Default()
	for i := 0; i < 7; i++ {
		cfg.Claude.RefreshInterval = 60 + i // force a real change each cycle, so each one backs up
		_, err := InstallClaude(p, bin, cfg, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}
	res, err := Uninstall("claude", p, t0.Add(10*time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, "bash mine.sh", gjson.Get(read(t, p), "statusLine.command").String(),
		"the original statusLine must survive despite the unrelated marker mention in permissions.allow")
}

func TestPaths(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	none := env(nil)
	require.Equal(t, "/h/.claude/settings.json", ClaudePath(none, "/h"))
	require.Equal(t, "/c/settings.json", ClaudePath(env(map[string]string{"CLAUDE_CONFIG_DIR": "/c"}), "/h"))
	require.Equal(t, "/h/.cursor/cli-config.json", CursorPath(none, "/h", "darwin"))
	require.Equal(t, "/x/cursor/cli-config.json", CursorPath(env(map[string]string{"XDG_CONFIG_HOME": "/x"}), "/h", "linux"))
	require.Equal(t, "/h/.cursor/cli-config.json", CursorPath(env(map[string]string{"XDG_CONFIG_HOME": "/x"}), "/h", "darwin"))
	require.Equal(t, "/d/cli-config.json", CursorPath(env(map[string]string{"CURSOR_CONFIG_DIR": "/d"}), "/h", "linux"))
	require.Equal(t, "/h/.codex/config.toml", CodexPath(none, "/h"))
	require.Equal(t, "/k/config.toml", CodexPath(env(map[string]string{"CODEX_HOME": "/k"}), "/h"))
}

func TestCheckBinary(t *testing.T) {
	require.NoError(t, checkBinary("/opt/homebrew/bin/agent-statusline", "/private/var/folders/xy/T"))
	require.Error(t, checkBinary("/private/var/folders/xy/T/go-build123/b001/exe/agent-statusline", "/private/var/folders/xy/T"))
	require.Error(t, checkBinary("/private/var/folders/xy/T/other/agent-statusline", "/private/var/folders/xy/T"))
	require.Error(t, checkBinary("relative/agent-statusline", "/tmp"))
}

func TestDetect(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
	look := func(name string) (string, error) {
		if name == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", os.ErrNotExist
	}
	require.Equal(t, map[string]bool{"claude": true, "cursor": false, "codex": true}, Detect(home, look))
}

// Fix round 4, IMPORTANT: pruning must protect only the single backup uninstall would restore
// each managed key from, so the backup count stays bounded by keepBackups + len(keys). With
// Claude subagents disabled, subagentStatusLine is absent (so "not ours") in every backup, which
// used to protect every backup and let the count grow without limit.
func TestBackup_BoundedWithClaudeSubagentsDisabled(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	cfg := config.Default()
	cfg.Claude.Subagents.Enabled = false
	for i := 0; i < 40; i++ {
		cfg.Claude.RefreshInterval = 60 + i // force a real change each cycle, so each one backs up
		_, err := InstallClaude(p, bin, cfg, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}
	bs, err := Backups(p)
	require.NoError(t, err)
	require.LessOrEqual(t, len(bs), keepBackups+2, "at most keepBackups plus one restore source per managed key")

	res, err := Uninstall("claude", p, t0.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, "bash /Users/me/.claude/statusline-command.sh", gjson.Get(read(t, p), "statusLine.command").String(),
		"the user's original statusLine must still be restored after 40 reinstalls")
}

func TestBackup_BoundedForCursor(t *testing.T) {
	p := write(t, "cli-config.json", `{"statusLine": {"type": "command", "command": "/usr/local/bin/mine"}}`+"\n")
	cfg := config.Default()
	for i := 0; i < 40; i++ {
		cfg.Cursor.UpdateIntervalMs = 300 + i // force a real change each cycle
		_, err := InstallCursor(p, bin, cfg, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}
	bs, err := Backups(p)
	require.NoError(t, err)
	require.LessOrEqual(t, len(bs), keepBackups+1)

	res, err := Uninstall("cursor", p, t0.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, "/usr/local/bin/mine", gjson.Get(read(t, p), "statusLine.command").String())
}

func TestBackup_BoundedForCodex(t *testing.T) {
	p := write(t, "config.toml", "[tui]\nstatus_line = [\"model-name\"]\n")
	cfg := config.Default()
	full := append([]string(nil), cfg.Codex.Items...)
	for i := 0; i < 40; i++ {
		cfg.Codex.Items = full
		if i%2 == 1 {
			cfg.Codex.Items = full[:len(full)-1] // alternate item lists, so each cycle changes the file
		}
		_, err := InstallCodex(p, cfg, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}
	bs, err := Backups(p)
	require.NoError(t, err)
	require.LessOrEqual(t, len(bs), keepBackups+1)

	res, err := Uninstall("codex", p, t0.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Contains(t, read(t, p), `status_line = ["model-name"]`)
}
