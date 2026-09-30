package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const bin = "/opt/homebrew/bin/agent-statusline"

var t0 = time.Unix(1790000000, 0)

const claudeSettings = `{
  "env": {
    "CLAUDE_CODE_DISABLE_MOUSE": "1"
  },
  "permissions": {
    "allow": ["Bash(git fetch:*)", "Read(//Users/me/**)"]
  },
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "notify.sh || true"}]}]
  },
  "model": "claude-fable-5-1[1m]",
  "effortLevel": "high"
}
`

const claudeWithUserStatusLine = `{
  "model": "x",
  "statusLine": {
    "type": "command",
    "command": "bash /Users/me/.claude/statusline-command.sh",
    "refreshInterval": 30000
  },
  "theme": "dark"
}
`

func write(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	return string(b)
}

func TestInstallClaude_OnlyStatusLineChanges(t *testing.T) {
	p := write(t, "settings.json", claudeSettings)
	res, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.NotEmpty(t, res.Backup)
	require.Equal(t, claudeSettings, read(t, res.Backup))

	out := read(t, p)
	require.True(t, json.Valid([]byte(out)))
	prefix := strings.TrimRight(claudeSettings[:strings.LastIndex(claudeSettings, "}")], " \n")
	require.True(t, strings.HasPrefix(out, prefix), "original bytes are untouched")
	require.Equal(t, bin+" render --harness claude", gjson.Get(out, "statusLine.command").String())
	require.Equal(t, "command", gjson.Get(out, "statusLine.type").String())
	require.EqualValues(t, 60, gjson.Get(out, "statusLine.refreshInterval").Int())
	require.Equal(t, bin+" render-subagents", gjson.Get(out, "subagentStatusLine.command").String())
	require.Equal(t, "notify.sh || true", gjson.Get(out, "hooks.Stop.0.hooks.0.command").String())
}

func TestInstallClaude_ReplacesInPlaceAndWarns(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	cfg := config.Default()
	cfg.Claude.Subagents.Enabled = false
	res, err := InstallClaude(p, bin, cfg, t0)
	require.NoError(t, err)
	old := gjson.Get(claudeWithUserStatusLine, "statusLine").Raw
	want := strings.Replace(claudeWithUserStatusLine, old,
		`{"type":"command","command":"`+bin+` render --harness claude","refreshInterval":60}`, 1)
	require.Equal(t, want, read(t, p))
	require.Contains(t, strings.Join(res.Warnings, "\n"), "statusline-command.sh")
}

func TestInstallClaude_Idempotent(t *testing.T) {
	p := write(t, "settings.json", claudeSettings)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	res, err := InstallClaude(p, bin, config.Default(), t0.Add(time.Second))
	require.NoError(t, err)
	require.False(t, res.Changed)
	bs, _ := Backups(p)
	require.Len(t, bs, 1)
}

func TestInstallClaude_MissingFileAndQuoting(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "settings.json")
	res, err := InstallClaude(p, "/Users/me/my tools/agent-statusline", config.Default(), t0)
	require.NoError(t, err)
	require.Empty(t, res.Backup)
	out := read(t, p)
	require.True(t, json.Valid([]byte(out)))
	require.Equal(t, `'/Users/me/my tools/agent-statusline' render --harness claude`, gjson.Get(out, "statusLine.command").String())
}

func TestInstallClaude_Refusals(t *testing.T) {
	p := write(t, "settings.json", "{nope")
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.ErrorContains(t, err, "not valid JSON")
	require.Equal(t, "{nope", read(t, p))

	p = write(t, "settings.json", `{"disableAllHooks": true}`)
	res, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "disableAllHooks")
}

func TestInstallCursor(t *testing.T) {
	p := write(t, "cli-config.json", `{"version": 1, "editor": {"vimMode": false}}`+"\n")
	res, err := InstallCursor(p, bin, config.Default(), t0)
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	require.Equal(t, bin+" render --harness cursor", gjson.Get(out, "statusLine.command").String())
	require.EqualValues(t, 0, gjson.Get(out, "statusLine.padding").Int())
	require.True(t, gjson.Get(out, "statusLine.padding").Exists())
	require.EqualValues(t, 300, gjson.Get(out, "statusLine.updateIntervalMs").Int())
	require.EqualValues(t, 1000, gjson.Get(out, "statusLine.timeoutMs").Int())

	_, err = InstallCursor(p, "/Users/me/my tools/agent-statusline", config.Default(), t0)
	require.ErrorContains(t, err, "space")
	_, err = InstallCursor(p, "agent-statusline", config.Default(), t0)
	require.ErrorContains(t, err, "absolute")
}

const codexConfig = "model = \"gpt-5.6-sol\"\n\n[tui]\ntheme = \"catppuccin-latte\"\n"

func TestInstallCodex(t *testing.T) {
	p := write(t, "config.toml", codexConfig)
	cfg := config.Default()
	cfg.Codex.Items = append(cfg.Codex.Items, "pr")
	res, err := InstallCodex(p, cfg, t0)
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, []string{"pr"}, res.Dropped)
	require.Contains(t, read(t, p), "[tui]\nstatus_line = [\"model-with-reasoning\"")

	cfg.Codex.Items = []string{"pr"}
	_, err = InstallCodex(p, cfg, t0)
	require.ErrorContains(t, err, "no segment")
}

// Final review, MINOR B: overwriting a subagentStatusLine or Codex status_line that isn't ours
// says so, the way statusLine already does.
func TestInstall_WarnsWhenReplacingForeignSubagentAndCodexLines(t *testing.T) {
	p := write(t, "settings.json", `{"subagentStatusLine": {"type": "command", "command": "mysub.sh"}}`+"\n")
	res, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "replaced your existing subagentStatusLine (mysub.sh); `uninstall claude` restores it")
	res, err = InstallClaude(p, bin, config.Default(), t0.Add(time.Second))
	require.NoError(t, err)
	require.Empty(t, res.Warnings, "reinstalling over our own lines is silent")

	c := write(t, "config.toml", "[tui]\nstatus_line = [\"model\"]\n")
	res, err = InstallCodex(c, config.Default(), t0)
	require.NoError(t, err)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "replaced your existing status_line ([\"model\"]); `uninstall codex` restores it")
	res, err = InstallCodex(c, config.Default(), t0.Add(time.Second))
	require.NoError(t, err)
	require.Empty(t, res.Warnings)
}
