package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestRunInstallAndUninstallList(t *testing.T) {
	dir := isolate(t)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(dir, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(dir, ".codex"))
	var out bytes.Buffer
	err := runInstallList(&out, []string{"claude", "codex"}, config.Default(), "/opt/homebrew/bin/agent-statusline")
	require.NoError(t, err)
	require.Contains(t, out.String(), "✓ claude")
	require.Contains(t, out.String(), "✓ codex")

	b, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	require.NoError(t, err)
	require.Contains(t, gjson.GetBytes(b, "statusLine.command").String(), "render --harness claude")

	out.Reset()
	require.NoError(t, runUninstallList(&out, []string{"claude", "codex", "cursor"}))
	b, _ = os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	require.False(t, gjson.GetBytes(b, "statusLine").Exists())
}

func TestInstallCmd_UnknownTool(t *testing.T) {
	isolate(t)
	_, _, err := execute(t, nil, "install", "vim")
	require.Error(t, err)
}
