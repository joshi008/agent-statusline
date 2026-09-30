package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit_NonInteractiveWritesDefault(t *testing.T) {
	dir := isolate(t)
	path := filepath.Join(dir, ".config", "agent-statusline", "config.yaml")
	out, _, err := execute(t, nil, "init")
	require.NoError(t, err)
	require.Contains(t, out, "Wrote "+path)
	data, _ := os.ReadFile(path)
	require.Contains(t, string(data), "# agent-statusline configuration")

	out, _, err = execute(t, nil, "init")
	require.NoError(t, err)
	require.Contains(t, out, "already exists")
}

func TestConfigEdit_NonInteractiveErrors(t *testing.T) {
	isolate(t)
	_, _, err := execute(t, nil, "config", "edit")
	require.ErrorContains(t, err, "interactive terminal")
}
