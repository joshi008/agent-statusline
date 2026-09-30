package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func execute(t *testing.T, stdin []byte, args ...string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	var out, errOut bytes.Buffer
	root.SetIn(bytes.NewReader(stdin))
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}

func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, ".cache"))
	return dir
}

func TestRenderCmd_Claude(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, fixtures.Read("claude-full"), "render", "--harness", "claude", "--no-color")
	require.NoError(t, err)
	require.Contains(t, out, "Fable 5.1")
	require.Contains(t, out, "ctx ▓▓▓▓░░░░░░ 37%")
}

func TestRenderCmd_EmptyStdinStillPrints(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, nil, "render", "--no-color")
	require.NoError(t, err)
	require.Equal(t, "?\n", out)
}

func TestRenderCmd_BrokenConfigUsesDefaults(t *testing.T) {
	dir := isolate(t)
	bad := filepath.Join(dir, "bad.yaml")
	require.NoError(t, writeFile(bad, "theme: [unclosed\n"))
	out, errOut, err := execute(t, fixtures.Read("claude-full"), "render", "--config", bad, "--no-color")
	require.NoError(t, err)
	require.Contains(t, out, "Fable 5.1")
	require.Contains(t, errOut, "using defaults")
}

func TestPreviewAndThemesCmd(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, nil, "preview", "--no-color", "--harness", "cursor")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "Cursor Grok 4.6 High · manual"), out)

	out, _, err = execute(t, nil, "themes", "--no-color")
	require.NoError(t, err)
	require.Contains(t, out, "default")
	require.Contains(t, out, "gradient")
	require.Contains(t, out, "powerline")
}

func TestPreviewCodex(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, nil, "preview", "--harness", "codex")
	require.NoError(t, err)
	require.Equal(t, "[tui]\nstatus_line = [\"model-with-reasoning\", \"context-used\", \"five-hour-limit\", \"weekly-limit\", \"estimated-thread-cost\", \"git-branch\"]\n", out)
}

func TestDoctorCmd_RunsInIsolation(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, nil, "doctor")
	require.NoError(t, err, out)
	require.Contains(t, out, "config")
	require.Contains(t, out, "speed")
}

// Final review, IMPORTANT 3: one typo'd key warns on stderr but keeps the user's theme.
func TestRenderCmd_UnknownKeyWarnsAndKeepsConfig(t *testing.T) {
	dir := isolate(t)
	good := filepath.Join(dir, "good.yaml")
	require.NoError(t, writeFile(good, "theme: gradient\n"))
	typo := filepath.Join(dir, "typo.yaml")
	require.NoError(t, writeFile(typo, "theme: gradient\ncolour: auto\n"))

	want, _, err := execute(t, fixtures.Read("claude-full"), "render", "--config", good)
	require.NoError(t, err)
	def, _, err := execute(t, fixtures.Read("claude-full"), "render")
	require.NoError(t, err)
	require.NotEqual(t, def, want, "the gradient theme renders differently from the default")

	out, errOut, err := execute(t, fixtures.Read("claude-full"), "render", "--config", typo)
	require.NoError(t, err)
	require.Equal(t, want, out, "the rest of the config (theme: gradient) still applies")
	require.Contains(t, errOut, "colour")
	require.NotContains(t, errOut, "using defaults")

	vout, _, err := execute(t, nil, "config", "validate", "--config", typo)
	require.Error(t, err, "config validate stays strict")
	require.Contains(t, vout, "colour")
}
