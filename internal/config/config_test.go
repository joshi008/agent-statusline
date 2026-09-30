package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

func ids(l Line) []string {
	var out []string
	for _, r := range l {
		out = append(out, r.ID)
	}
	return out
}

func TestDefault(t *testing.T) {
	c := Default()
	require.Equal(t, 1, c.Version)
	require.Equal(t, "default", c.Theme)
	require.Equal(t, "auto", c.Color)
	require.Equal(t, 50.0, c.Style.Thresholds.Warn)
	require.Equal(t, 80.0, c.Style.Thresholds.Crit)
	require.True(t, *c.Style.Countdown)
	require.Nil(t, c.Style.Emoji)
	require.Len(t, c.Claude.Lines, 2)
	require.Equal(t, "session", c.Claude.Lines[0][0].ID)
	require.Equal(t, 60, c.Claude.RefreshInterval)
	require.True(t, c.Claude.Subagents.Enabled)
	require.Equal(t, 1000, c.Cursor.TimeoutMs)
	require.Len(t, c.Codex.Items, 7)
}

func TestDefaultMatchesTwoLinePreset(t *testing.T) {
	c := Default()
	p := Default()
	require.NoError(t, ApplyPreset(p, "two-line"))
	require.Equal(t, c.Claude.Lines, p.Claude.Lines)
	require.Equal(t, c.Cursor.Lines, p.Cursor.Lines)
}

func TestParse_PartialOverrideKeepsDefaults(t *testing.T) {
	c, err := Parse([]byte("style:\n  thresholds: { warn: 60 }\nclaude:\n  lines:\n    - [model, context]\n"))
	require.NoError(t, err)
	require.Equal(t, 60.0, c.Style.Thresholds.Warn)
	require.Equal(t, 80.0, c.Style.Thresholds.Crit)
	require.Len(t, c.Claude.Lines, 1)
	require.Equal(t, []string{"model", "context"}, ids(c.Claude.Lines[0]))
	require.Equal(t, 60, c.Claude.RefreshInterval)
	require.Len(t, c.Cursor.Lines, 2)
}

func TestParse_UnknownKeyFails(t *testing.T) {
	_, err := Parse([]byte("colour: auto\n"))
	require.ErrorContains(t, err, "colour")
}

func TestParse_Empty(t *testing.T) {
	c, err := Parse(nil)
	require.NoError(t, err)
	require.Equal(t, Default(), c)
}

func TestSegRef_MapForm(t *testing.T) {
	c, err := Parse([]byte("claude:\n  lines:\n    - [model, { id: context, style: tokens }, cmd:k8s]\n"))
	require.NoError(t, err)
	l := c.Claude.Lines[0]
	require.Equal(t, "context", l[1].ID)
	require.Equal(t, "tokens", l[1].Opts["style"])
	require.Equal(t, "cmd:k8s", l[2].ID)
	require.Nil(t, l[0].Opts)
}

func TestSegRef_Invalid(t *testing.T) {
	_, err := Parse([]byte("claude:\n  lines:\n    - [{ style: tokens }]\n"))
	require.ErrorContains(t, err, "needs an id")
	_, err = Parse([]byte("claude:\n  lines:\n    - [[model]]\n"))
	require.Error(t, err)
}

func TestCustomTTL(t *testing.T) {
	c, err := Parse([]byte("custom:\n  - { id: k8s, run: \"kubectl config current-context\", ttl: 60s }\n"))
	require.NoError(t, err)
	cu, ok := c.FindCustom("k8s")
	require.True(t, ok)
	require.Equal(t, 60*time.Second, cu.TTL)
}

func TestLoad_MissingFileIsDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	require.Equal(t, Default(), c)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg", "config.yaml")
	c := Default()
	c.Theme = "gradient"
	c.Claude.Lines = []Line{{{ID: "model"}, {ID: "context", Opts: map[string]any{"style": "pct"}}}}
	c.Custom = []Custom{{ID: "env", Text: "prod", TTL: 30 * time.Second}}
	require.NoError(t, Save(p, c))
	got, err := Load(p)
	require.NoError(t, err)
	require.Equal(t, c, got)
}

func TestClone_Independent(t *testing.T) {
	a := Default()
	b := a.Clone()
	b.Claude.Lines[0][0].ID = "changed"
	require.Equal(t, "session", a.Claude.Lines[0][0].ID)
}

func TestDefaultPath(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	require.Equal(t, "/x/agent-statusline/config.yaml", DefaultPath(env(map[string]string{"XDG_CONFIG_HOME": "/x"}), "/h"))
	require.Equal(t, "/h/.config/agent-statusline/config.yaml", DefaultPath(env(nil), "/h"))
}

func TestLinesForAndSegmentOpts(t *testing.T) {
	c, err := Parse([]byte("segments:\n  dir: { depth: 2 }\n"))
	require.NoError(t, err)
	require.Equal(t, c.Cursor.Lines, c.LinesFor(model.HarnessCursor))
	require.Equal(t, c.Claude.Lines, c.LinesFor(model.HarnessClaude))
	require.Equal(t, 2, c.SegmentOpts("dir")["depth"])
	require.Nil(t, c.SegmentOpts("model"))
}

func TestApplyPreset(t *testing.T) {
	c := Default()
	require.NoError(t, ApplyPreset(c, "minimal"))
	require.Equal(t, []string{"model", "context", "limit_5h"}, ids(c.Claude.Lines[0]))
	require.Error(t, ApplyPreset(c, "nope"))
}

func TestDefaultYAMLIsCommented(t *testing.T) {
	require.Contains(t, string(DefaultYAML()), "# agent-statusline configuration")
	_, err := os.Stat("default.yaml")
	require.NoError(t, err)
}

// Final review, IMPORTANT 3: at render time an unknown key is a warning, not a reason to throw
// the rest of the user's config away.
func TestLoadLenient_UnknownKeysWarnAndKeepTheRest(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte("theme: gradient\ncolour: auto\nclaude:\n  refrsh: 1\n"), 0o644))
	c, warnings, err := LoadLenient(p)
	require.NoError(t, err)
	require.Equal(t, "gradient", c.Theme)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "colour")
	require.Contains(t, warnings[1], "refrsh")

	_, err = Load(p)
	require.Error(t, err, "Load stays strict")
}

func TestLoadLenient_RealErrorsStillFail(t *testing.T) {
	dir := t.TempDir()
	syntax := filepath.Join(dir, "syntax.yaml")
	require.NoError(t, os.WriteFile(syntax, []byte("theme: [unclosed\n"), 0o644))
	_, _, err := LoadLenient(syntax)
	require.Error(t, err)

	typed := filepath.Join(dir, "typed.yaml")
	require.NoError(t, os.WriteFile(typed, []byte("colour: auto\nclaude:\n  lines: 7\n"), 0o644))
	_, _, err = LoadLenient(typed)
	require.Error(t, err, "a type error is not just an unknown key")

	c, warnings, err := LoadLenient(filepath.Join(dir, "missing.yaml"))
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, Default(), c)
}
