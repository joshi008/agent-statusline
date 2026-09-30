package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func testEnv(extra map[string]string) func(string) string {
	m := map[string]string{"HOME": "/Users/example"}
	for k, v := range extra {
		m[k] = v
	}
	return func(k string) string { return m[k] }
}

func plainInput(fixture, harness string) Input {
	return Input{Payload: fixtures.Read(fixture), Harness: harness, Config: config.Default(), Getenv: testEnv(nil),
		NoColor: true, Now: PreviewNow, DisableCache: true, Run: PreviewRunner}
}

func TestRender_ClaudeFullTwoLinePlain(t *testing.T) {
	out := Render(plainInput("claude-full", "claude"))
	require.Equal(t,
		"status-bar research · Fable 5.1 · high · think · status-bar · main · +2 ~1 ↑1 · #42\n"+
			"ctx ▓▓▓▓░░░░░░ 37% · $1.23 · 5h ▓▓▓▓░░░░░░ 42% ~23m · 7d ▓▓▓▓▓▓▓░░░ 71% ~3d11h · cache 91%\n",
		out)
}

func TestRender_CursorFullTwoLinePlain(t *testing.T) {
	out := Render(plainInput("cursor-full", "cursor"))
	require.Equal(t,
		"Cursor Grok 4.6 High · manual · status-bar · main · +2 ~1 ↑1 · #7 ✓\n"+
			"ctx ▓▓▓░░░░░░░ 31%\n",
		out)
}

func TestRender_AutoDetect(t *testing.T) {
	in := plainInput("cursor-minimal", "auto")
	out := Render(in)
	require.True(t, strings.HasPrefix(out, "Auto · max · auto · "), out)
	require.Equal(t, 1, strings.Count(out, "\n"), "empty second line is dropped")
}

func TestRender_MinimalPayloadNeverShowsFakeZero(t *testing.T) {
	out := Render(plainInput("claude-minimal", "claude"))
	require.Contains(t, out, "Fable 5.1")
	require.NotContains(t, out, "0%")
	require.NotContains(t, out, "NaN")
}

func TestRender_GarbagePayload(t *testing.T) {
	in := plainInput("claude-full", "claude")
	in.Payload = []byte("not json")
	require.Equal(t, "?\n", Render(in))
	in.Payload = nil
	require.Equal(t, "?\n", Render(in))
}

func TestRender_NoLinesFallsBackToModel(t *testing.T) {
	in := plainInput("claude-full", "claude")
	in.Config.Claude.Lines = nil
	require.Equal(t, "Fable 5.1\n", Render(in))
}

func TestRender_UnknownThemeAndSegmentDegrade(t *testing.T) {
	in := plainInput("claude-full", "claude")
	in.Config.Theme = "nope"
	in.Config.Claude.Lines = []config.Line{{{ID: "bogus"}, {ID: "model"}, {ID: "cost"}}}
	var logs []string
	in.Logf = func(f string, a ...any) { logs = append(logs, f) }
	require.Equal(t, "Fable 5.1 · $1.23\n", Render(in))
	require.NotEmpty(t, logs)
}

func TestRender_WidthFitting(t *testing.T) {
	in := plainInput("claude-full", "claude")
	in.Width = 40
	for _, line := range strings.Split(strings.TrimSuffix(Render(in), "\n"), "\n") {
		require.LessOrEqual(t, len([]rune(line)), 40, line)
	}
}

func TestRender_StyleOverridesAndColor(t *testing.T) {
	in := plainInput("claude-full", "claude")
	sep := " | "
	in.Config.Style.Separator = &sep
	in.Config.Claude.Lines = []config.Line{{{ID: "model"}, {ID: "cost"}}}
	require.Equal(t, "Fable 5.1 | $1.23\n", Render(in))

	in.NoColor = false
	in.Getenv = testEnv(map[string]string{"COLORTERM": "truecolor"})
	require.Contains(t, Render(in), "\x1b[1m", "model is bold in the default theme")
	in.Getenv = testEnv(map[string]string{"COLORTERM": "truecolor", "NO_COLOR": "1"})
	require.NotContains(t, Render(in), "\x1b[")
}

func TestColorLevel(t *testing.T) {
	env := testEnv(map[string]string{"TERM": "xterm-256color"})
	require.Equal(t, 2, int(ColorLevel("auto", false, env)))
	require.Equal(t, 3, int(ColorLevel("truecolor", false, env)))
	require.Equal(t, 0, int(ColorLevel("truecolor", true, env)))
	require.Equal(t, 2, int(ColorLevel("bogus", false, env)))
}

func TestRender_Budget(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test")
	}
	in := plainInput("claude-full", "claude")
	const n = 200
	start := time.Now()
	for i := 0; i < n; i++ {
		Render(in)
	}
	avg := time.Since(start) / n
	require.Less(t, avg, 10*time.Millisecond, "average render %s exceeds the 10 ms budget", avg)
}

func BenchmarkRenderClaudeFull(b *testing.B) {
	in := plainInput("claude-full", "claude")
	in.NoColor = false
	in.Getenv = testEnv(map[string]string{"COLORTERM": "truecolor"})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Render(in)
	}
}
