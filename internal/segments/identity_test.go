package segments

import (
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

func TestIdentitySegments(t *testing.T) {
	s := &model.Snapshot{
		SessionName: "research", Version: "2.1.283", Effort: "xhigh",
		Model:    model.ModelInfo{ID: "claude-fable-5-1[1m]", DisplayName: "Fable 5.1", MaxMode: true},
		Thinking: ptr(true), FastMode: ptr(true), Autorun: ptr(false),
		Agent: "reviewer", Vim: "NORMAL", Now: time.Date(2026, 9, 29, 14, 5, 0, 0, time.UTC),
	}
	cases := []struct {
		id   string
		opts Options
		want string
	}{
		{"session", nil, "research"},
		{"model", nil, "Fable 5.1"},
		{"model", Options{"show_id": true}, "Fable 5.1 (claude-fable-5-1[1m])"},
		{"effort", nil, "xhigh"},
		{"effort", Options{"format": "eff:{level}"}, "eff:xhigh"},
		{"thinking", nil, "think"},
		{"fast", nil, "fast"},
		{"max_mode", nil, "max"},
		{"autorun", nil, "manual"},
		{"agent", nil, "agent:reviewer"},
		{"vim", nil, "NORMAL"},
		{"version", nil, "v2.1.283"},
		{"time", nil, "14:05"},
		{"time", Options{"format": "15:04:05"}, "14:05:00"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, run(t, c.id, newEnv(t, s, c.opts)), c.id)
	}
}

func TestIdentitySegments_AbsentSkip(t *testing.T) {
	s := &model.Snapshot{Thinking: ptr(false), FastMode: ptr(false)}
	for _, id := range []string{"session", "model", "effort", "thinking", "fast", "max_mode", "autorun", "agent", "vim", "version"} {
		d, _ := Lookup(id)
		require.Nil(t, d.Render(newEnv(t, s, nil)), id)
	}
}

func TestModel_FallsBackToID(t *testing.T) {
	s := &model.Snapshot{Model: model.ModelInfo{ID: "grok-4.6"}}
	require.Equal(t, "grok-4.6", run(t, "model", newEnv(t, s, nil)))
}

func TestAutorunOn(t *testing.T) {
	s := &model.Snapshot{Autorun: ptr(true)}
	require.Equal(t, "auto", run(t, "autorun", newEnv(t, s, nil)))
}

func TestCustomSegments(t *testing.T) {
	s := &model.Snapshot{Custom: map[string]string{"k8s": "prod-eu", "env": "prod"}}
	require.Equal(t, "prod-eu", run(t, "cmd:k8s", newEnv(t, s, nil)))
	require.Equal(t, "prod", run(t, "text:env", newEnv(t, s, nil)))
	d, ok := Lookup("cmd:missing")
	require.True(t, ok)
	require.Nil(t, d.Render(newEnv(t, s, nil)))
	_, ok = Lookup("cmd:")
	require.False(t, ok)
	_, ok = Lookup("bogus")
	require.False(t, ok)
	require.True(t, Known("text:anything"))
}

func TestOptions(t *testing.T) {
	o := Options{"s": "x", "i": 3, "f": 2.0, "b": true}
	require.Equal(t, "x", o.String("s", "d"))
	require.Equal(t, "d", o.String("missing", "d"))
	require.Equal(t, 3, o.Int("i", 0))
	require.Equal(t, 2, o.Int("f", 0))
	require.Equal(t, 7, o.Int("s", 7))
	require.True(t, o.Bool("b", false))
	m := MergeOptions(map[string]any{"a": 1, "b": 1}, map[string]any{"b": 2})
	require.Equal(t, Options{"a": 1, "b": 2}, m)
}

func TestHumanizeAndLevelRole(t *testing.T) {
	require.Equal(t, "950", Humanize(950))
	require.Equal(t, "4.1k", Humanize(4100))
	require.Equal(t, "372k", Humanize(372000))
	require.Equal(t, "1M", Humanize(1_000_000))
	require.Equal(t, "ok", LevelRole(49.9, 50, 80))
	require.Equal(t, "warn", LevelRole(50, 50, 80))
	require.Equal(t, "crit", LevelRole(80, 50, 80))
}
