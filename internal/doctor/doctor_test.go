package doctor

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func deps(t *testing.T) (Deps, string) {
	home := t.TempDir()
	return Deps{
		Getenv: func(key string) string {
			return map[string]string{"COLORTERM": "truecolor", "TERM_PROGRAM": "Apple_Terminal"}[key]
		},
		Home:         home,
		GOOS:         "darwin",
		ConfigPath:   filepath.Join(home, ".config", "agent-statusline", "config.yaml"),
		LookPath:     func(string) (string, error) { return "", errors.New("not found") },
		Stat:         func(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist },
		CodexVersion: func() (string, error) { return "", errors.New("no codex") },
		RenderTime:   func() time.Duration { return time.Millisecond },
	}, home
}

func put(t *testing.T, home, relative, content string) {
	path := filepath.Join(home, relative)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func find(checks []Check, name string) []Check {
	var out []Check
	for _, check := range checks {
		if check.Name == name {
			out = append(out, check)
		}
	}
	return out
}

func details(checks []Check) string {
	var out strings.Builder
	for _, check := range checks {
		out.WriteString(check.Status.Symbol() + " " + check.Detail + "\n")
	}
	return out.String()
}

func TestDoctor_FreshMachineHasNoFailures(t *testing.T) {
	d, _ := deps(t)
	checks := Run(d)
	require.Zero(t, Failed(checks), details(checks))
	require.Equal(t, Warn, find(checks, "binary")[0].Status)
	require.Equal(t, OK, find(checks, "config")[0].Status)
	require.Equal(t, Warn, find(checks, "claude")[0].Status)
	require.Equal(t, Warn, find(checks, "codex")[0].Status)
	require.Contains(t, find(checks, "links")[0].Detail, "Apple_Terminal")
}

func TestDoctor_ClaudeRefreshIntervalInSeconds(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".claude/settings.json",
		`{"statusLine":{"type":"command","command":"bash ~/.claude/statusline-command.sh","refreshInterval":30000}}`)
	out := details(find(Run(d), "claude"))
	require.Contains(t, out, "not agent-statusline")
	require.Contains(t, out, "rate-limit countdowns will go stale")
}

func TestDoctor_ClaudeOursBinaryMissingAndPresent(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".claude/settings.json",
		`{"statusLine":{"type":"command","command":"/opt/homebrew/bin/agent-statusline render --harness claude","refreshInterval":60}}`)
	require.Equal(t, Fail, find(Run(d), "claude")[0].Status)
	d.Stat = func(string) (fs.FileInfo, error) { return nil, nil }
	require.Equal(t, OK, find(Run(d), "claude")[0].Status)
}

func TestDoctor_DisableAllHooks(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".claude/settings.json", `{"disableAllHooks":true}`)
	require.Contains(t, details(find(Run(d), "claude")), "disableAllHooks")
	require.Positive(t, Failed(Run(d)))
}

func TestDoctor_CodexItemsAndVersion(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".codex/config.toml", "[tui]\nstatus_line = [\"model\", \"limit_7d\"]\n")
	d.CodexVersion = func() (string, error) { return "codex-cli 0.160.0", nil }
	out := details(find(Run(d), "codex"))
	require.Contains(t, out, "✗ unknown Codex items: limit_7d")
	require.Contains(t, out, "verified against 0.159.2")

	put(t, home, ".codex/config.toml", "[tui]\nstatus_line = [\"model\"] # agent-statusline\n")
	d.CodexVersion = func() (string, error) { return "codex-cli 0.159.2", nil }
	checks := find(Run(d), "codex")
	require.Len(t, checks, 1)
	require.Equal(t, OK, checks[0].Status)
}

func TestDoctor_CodexStatusLineNotOurs(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".codex/config.toml", "[tui]\nstatus_line = [\"model\"]\n")
	out := details(find(Run(d), "codex"))
	require.Contains(t, out, "! tui.status_line = [\"model\"] was not set by agent-statusline")
	require.Contains(t, out, "run `agent-statusline install codex`")
}

func TestDoctor_BadConfigAndSlowRender(t *testing.T) {
	d, home := deps(t)
	put(t, home, ".config/agent-statusline/config.yaml", "claude:\n  lines: [[bogus]]\n")
	d.RenderTime = func() time.Duration { return 20 * time.Millisecond }
	checks := Run(d)
	require.Equal(t, Fail, find(checks, "config")[0].Status)
	require.Equal(t, Warn, find(checks, "speed")[0].Status)
}

func TestFirstTokenHandlesInstallerShellQuoting(t *testing.T) {
	require.Equal(t, "/Users/me/my tools/agent-statusline", firstToken(`'/Users/me/my tools/agent-statusline' render`))
	require.Equal(t, "/Users/o'connor/agent-statusline", firstToken(`'/Users/o'\''connor/agent-statusline' render`))
}
