package engine

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden file; run: go test ./internal/engine -run TestGolden -update")
	require.Equal(t, string(want), got, name)
}

func TestGolden_Plain(t *testing.T) {
	for _, fx := range fixtures.Names {
		for _, preset := range config.PresetNames {
			for _, th := range []string{"default", "gradient"} {
				cfg := config.Default()
				require.NoError(t, config.ApplyPreset(cfg, preset))
				cfg.Theme = th
				h := strings.SplitN(fx, "-", 2)[0]
				out := Render(Input{Payload: fixtures.Read(fx), Harness: h, Config: cfg, Getenv: testEnv(nil),
					NoColor: true, Now: PreviewNow, DisableCache: true, Run: PreviewRunner})
				golden(t, fmt.Sprintf("%s__%s__%s.txt", fx, preset, th), out)
			}
		}
	}
}

func TestGolden_ANSI(t *testing.T) {
	for _, th := range []string{"default", "gradient"} {
		cfg := config.Default()
		cfg.Theme = th
		out := Render(Input{Payload: fixtures.Read("claude-full"), Harness: "claude", Config: cfg,
			Getenv: testEnv(map[string]string{"COLORTERM": "truecolor"}), Now: PreviewNow, DisableCache: true, Run: PreviewRunner})
		golden(t, "claude-full__two-line__"+th+".ansi", out)
	}
}
