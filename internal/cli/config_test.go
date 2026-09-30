package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigPathAndInitFile(t *testing.T) {
	dir := isolate(t)
	want := filepath.Join(dir, ".config", "agent-statusline", "config.yaml")

	out, _, err := execute(t, nil, "config", "path")
	require.NoError(t, err)
	require.Equal(t, want+" (not created yet)\n", out)

	out, _, err = execute(t, nil, "config", "init-file")
	require.NoError(t, err)
	require.Contains(t, out, want)
	b, _ := os.ReadFile(want)
	require.Contains(t, string(b), "# agent-statusline configuration")

	_, _, err = execute(t, nil, "config", "init-file")
	require.ErrorContains(t, err, "--force")
	require.NoError(t, os.WriteFile(want, []byte("theme: gradient\n"), 0o644))
	_, _, err = execute(t, nil, "config", "init-file", "--force")
	require.NoError(t, err)
	b, _ = os.ReadFile(want)
	require.Contains(t, string(b), "theme: default")
}

func TestConfigValidate(t *testing.T) {
	dir := isolate(t)
	out, _, err := execute(t, nil, "config", "validate")
	require.NoError(t, err)
	require.Contains(t, out, "no config file")

	p := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte("claude:\n  lines: [[model, bogus]]\ncodex:\n  items: [model, pr]\n"), 0o644))
	out, _, err = execute(t, nil, "config", "validate", "--config", p)
	require.Error(t, err)
	require.Contains(t, out, `unknown segment "bogus"`)
	require.Contains(t, out, "codex: no item for pr")

	require.NoError(t, os.WriteFile(p, []byte("colour: auto\n"), 0o644))
	out, _, err = execute(t, nil, "config", "validate", "--config", p)
	require.Error(t, err)
	require.Contains(t, out, "colour")

	require.NoError(t, os.WriteFile(p, []byte("theme: gradient\n"), 0o644))
	out, _, err = execute(t, nil, "config", "validate", "--config", p)
	require.NoError(t, err)
	require.Contains(t, out, "✓")
}

func TestConfigShow(t *testing.T) {
	isolate(t)
	out, _, err := execute(t, nil, "config", "show")
	require.NoError(t, err)
	require.True(t, strings.Contains(out, "theme: default"), out)
	require.Contains(t, out, "limit_5h")
}
