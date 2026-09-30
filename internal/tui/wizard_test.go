package tui

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSetLinesKeepsInlineOptions(t *testing.T) {
	cfg := config.Default()
	cfg.Claude.Lines = []config.Line{{{ID: "model"}, {ID: "context", Opts: map[string]any{"style": "pct"}}}}
	setLines(cfg, "claude", [][]string{{"context"}, {"model", "cost"}})
	require.Equal(t, []config.Line{
		{{ID: "context", Opts: map[string]any{"style": "pct"}}},
		{{ID: "model"}, {ID: "cost"}},
	}, cfg.Claude.Lines)
	require.Equal(t, [][]string{{"context"}, {"model", "cost"}}, idsOf(cfg.Claude.Lines))

	setLines(cfg, "codex", [][]string{{"model", "branch"}})
	require.Equal(t, []string{"model", "branch"}, cfg.Codex.Items)
	setLines(cfg, "codex", nil)
	require.Empty(t, cfg.Codex.Items)
}

func TestApplyLook(t *testing.T) {
	cfg := config.Default()
	require.NoError(t, applyLook(cfg, "keep", "gradient"))
	require.Len(t, cfg.Claude.Lines, 2)
	require.Equal(t, "gradient", cfg.Theme)
	require.NoError(t, applyLook(cfg, "minimal", "default"))
	require.Len(t, cfg.Claude.Lines, 1)
	require.Error(t, applyLook(cfg, "nope", "default"))
}
