package engine

import (
	"regexp"
	"strings"
	"testing"

	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/theme"
	"github.com/stretchr/testify/require"
)

func defaultTheme(t *testing.T) *theme.Theme {
	th, err := theme.Resolve("default", nil)
	require.NoError(t, err)
	return th
}

func TestLine_NoDanglingSeparator(t *testing.T) {
	th := defaultTheme(t)
	parts := []Part{{"model", "A"}, {"limit_5h", ""}, {"cost", "$1"}, {"limit_7d", ""}}
	require.Equal(t, "A · $1", JoinLine(parts, th, render.LevelNone))
	require.Equal(t, "", JoinLine([]Part{{"x", ""}}, th, render.LevelNone))
	require.Equal(t, "", FitLine(nil, th, render.LevelNone, 80))
}

func TestFit_DropOrder(t *testing.T) {
	th := defaultTheme(t)
	parts := []Part{{"model", "Fable 5.1"}, {"version", "v2.1.283"}, {"cost", "$1.23"}}
	require.Equal(t, "Fable 5.1 · v2.1.283 · $1.23", FitLine(parts, th, render.LevelNone, 0), "width 0 = no fitting")
	require.Equal(t, "Fable 5.1 · $1.23", FitLine(parts, th, render.LevelNone, 17))
	require.Equal(t, "Fable 5.1", FitLine(parts, th, render.LevelNone, 9))
	require.Equal(t, "Fable…", FitLine(parts, th, render.LevelNone, 6), "last survivor is truncated")
}

func TestFit_Narrow60(t *testing.T) {
	th := defaultTheme(t)
	parts := []Part{
		{"session", "status-bar research"}, {"model", "Fable 5.1"}, {"effort", "high"}, {"thinking", "think"},
		{"dir", "status-bar"}, {"branch", "main"}, {"git_status", "+2 ~1 ↑1"}, {"pr", "#42"},
		{"context", "ctx ▓▓▓▓░░░░░░ 37%"}, {"cost", "$1.23"}, {"limit_5h", "5h ▓▓▓▓░░░░░░ 42% ~23m"},
	}
	out := FitLine(parts, th, render.LevelNone, 60)
	require.LessOrEqual(t, render.Width(out), 60)
	require.Contains(t, out, "Fable 5.1")
	require.Contains(t, out, "ctx ▓▓▓▓░░░░░░ 37%")
	require.NotContains(t, out, "think", "thinking drops before context and model")
}

var sgrOrOSC = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\]8;;[^\x07]*\x07`)

func TestFit_NeverSplitsEscape(t *testing.T) {
	th := defaultTheme(t)
	red, _ := render.ParseStyle("#ff0000 bold")
	long := red.Paint(strings.Repeat("x", 50), render.LevelTrue)
	link := "\x1b]8;;https://x/pull/1\x07" + red.Paint("#1", render.LevelTrue) + "\x1b]8;;\x07"
	for _, w := range []int{1, 5, 20, 49} {
		out := FitLine([]Part{{"model", long}, {"pr", link}}, th, render.LevelTrue, w)
		require.LessOrEqual(t, render.Width(out), w)
		require.NotContains(t, sgrOrOSC.ReplaceAllString(out, ""), "\x1b", "width %d left a partial escape", w)
	}
}

func TestJoinLine_PaintsSeparatorAndCaps(t *testing.T) {
	th := defaultTheme(t)
	out := JoinLine([]Part{{"a", "A"}, {"b", "B"}}, th, render.LevelBasic)
	require.Equal(t, "A\x1b[2m · \x1b[0mB", out)
	th.Caps = [2]string{"[", "]"}
	require.Equal(t, "[A · B]", JoinLine([]Part{{"a", "A"}, {"b", "B"}}, th, render.LevelNone))
}
