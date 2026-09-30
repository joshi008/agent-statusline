package segments

import (
	"strings"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/theme"
	"github.com/stretchr/testify/require"
)

func claudeSnap() *model.Snapshot {
	return &model.Snapshot{
		Now:     time.Unix(1790000000, 0),
		Context: &model.ContextWindow{UsedPct: ptr(37.2), Size: ptr(int64(1_000_000)), TotalInput: 372000, TotalOutput: 4100},
		Cost:    &model.Cost{USD: 1.2345, DurationMs: 2712000, APIDurationMs: 640000, LinesAdded: 156, LinesRemoved: 23},
		Cache:   &model.PromptCache{Warm: true, HitRatio: ptr(0.91)},
		Limits: map[string]model.Limit{
			"5h":    {UsedPct: 42, ResetsAt: ptr(int64(1790001400))},
			"7d":    {UsedPct: 71, ResetsAt: ptr(int64(1790300000))},
			"spend": {UsedPct: 120.5},
		},
	}
}

func TestContext(t *testing.T) {
	s := claudeSnap()
	require.Equal(t, "ctx ▓▓▓▓░░░░░░ 37%", run(t, "context", newEnv(t, s, nil)))
	require.Equal(t, "ctx 37%", run(t, "context", newEnv(t, s, Options{"style": "pct"})))
	require.Equal(t, "ctx ▓▓▓▓░░░░░░", run(t, "context", newEnv(t, s, Options{"style": "bar"})))
	require.Equal(t, "ctx 372k/1M", run(t, "context", newEnv(t, s, Options{"style": "tokens"})))
	require.Equal(t, "ctx ▓▓▓▓░░░░░░ 372k/1M", run(t, "context", newEnv(t, s, Options{"style": "bar+tokens"})))
	require.Equal(t, "▓▓▓▓░░░░░░ 37%", run(t, "context", newEnv(t, s, Options{"label": ""})))
	require.Equal(t, "ctx ▓▓░░░ 37%", run(t, "context", newEnv(t, s, Options{"bar_width": 5})))
}

func TestContext_NullPct(t *testing.T) {
	s := claudeSnap()
	s.Context.UsedPct = nil
	require.Nil(t, mustLookup(t, "context").Render(newEnv(t, s, nil)), "null used_percentage must skip, never print 0%")
	s.Context = nil
	require.Nil(t, mustLookup(t, "context").Render(newEnv(t, s, nil)))
}

func TestContext_RoleAndEmoji(t *testing.T) {
	s := claudeSnap()
	s.Context.UsedPct = ptr(85.0)
	cells := mustLookup(t, "context").Render(newEnv(t, s, nil))
	require.Equal(t, "crit", cells[len(cells)-1].Role)

	e := newEnv(t, s, nil)
	th, err := theme.Resolve("gradient", nil)
	require.NoError(t, err)
	e.Theme = th
	s.Context.UsedPct = ptr(37.0)
	out := text(mustLookup(t, "context").Render(e))
	require.True(t, strings.HasPrefix(out, "ctx ⚡ "), out)
	require.Contains(t, out, strings.Repeat("█", 7)) // 20-wide bar, 37% → 7 blocks
}

func TestLimits(t *testing.T) {
	s := claudeSnap()
	require.Equal(t, "5h ▓▓▓▓░░░░░░ 42% ~23m", run(t, "limit_5h", newEnv(t, s, nil)))
	require.Equal(t, "7d ▓▓▓▓▓▓▓░░░ 71% ~3d11h", run(t, "limit_7d", newEnv(t, s, nil)))
	require.Equal(t, "5h ▓▓▓▓░░░░░░ 42%", run(t, "limit_5h", newEnv(t, s, Options{"countdown": false})))
	require.Equal(t, "5h 42% ~23m", run(t, "limit_5h", newEnv(t, s, Options{"style": "pct"})))
	require.Equal(t, "spend 121%", run(t, "spend", newEnv(t, s, nil)))

	e := newEnv(t, s, nil)
	e.Countdown = false
	require.Equal(t, "5h ▓▓▓▓░░░░░░ 42%", run(t, "limit_5h", e))
}

func TestLimit_PastResetHidesCountdown(t *testing.T) {
	s := claudeSnap()
	s.Now = time.Unix(1790001500, 0)
	require.Equal(t, "5h ▓▓▓▓░░░░░░ 42%", run(t, "limit_5h", newEnv(t, s, nil)))
}

func TestLimit_MissingWindowSkips(t *testing.T) {
	s := claudeSnap()
	delete(s.Limits, "7d")
	require.Nil(t, mustLookup(t, "limit_7d").Render(newEnv(t, s, nil)))
}

func TestCostDurationLines(t *testing.T) {
	s := claudeSnap()
	require.Equal(t, "$1.23", run(t, "cost", newEnv(t, s, nil)))
	require.Equal(t, "1.234 USD", run(t, "cost", newEnv(t, s, Options{"format": "%.3f USD"})))
	require.Equal(t, "45m12s", run(t, "duration", newEnv(t, s, nil)))
	require.Equal(t, "45m12s (api 10m40s)", run(t, "duration", newEnv(t, s, Options{"api": true})))
	require.Equal(t, "+156 -23", run(t, "lines", newEnv(t, s, nil)))

	s.Cost = &model.Cost{}
	require.Equal(t, "$0.00", run(t, "cost", newEnv(t, s, nil)), "a real zero cost is shown")
	require.Nil(t, mustLookup(t, "duration").Render(newEnv(t, s, nil)))
	require.Nil(t, mustLookup(t, "lines").Render(newEnv(t, s, nil)))
	s.Cost = nil
	require.Nil(t, mustLookup(t, "cost").Render(newEnv(t, s, nil)))
}

func TestCacheAndTokens(t *testing.T) {
	s := claudeSnap()
	require.Equal(t, "cache 91%", run(t, "cache", newEnv(t, s, nil)))
	s.Cache.Warm = false
	require.Equal(t, "cache cold", run(t, "cache", newEnv(t, s, nil)))
	require.Equal(t, "in 372k out 4.1k", run(t, "tokens", newEnv(t, s, nil)))
	s.Context = &model.ContextWindow{}
	require.Nil(t, mustLookup(t, "tokens").Render(newEnv(t, s, nil)))
}

func TestCatalogComplete(t *testing.T) {
	require.Len(t, All(), len(CatalogOrder))
	require.Len(t, registry, len(CatalogOrder))
	for _, d := range All() {
		require.NotEmpty(t, d.Desc, d.ID)
		require.True(t, d.Claude || d.Cursor, d.ID)
	}
}

func mustLookup(t *testing.T, id string) *Def {
	t.Helper()
	d, ok := Lookup(id)
	require.True(t, ok, id)
	return d
}
