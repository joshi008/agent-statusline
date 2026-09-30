package theme

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinsParseAndResolve(t *testing.T) {
	require.Len(t, Builtins(), 3)
	for _, n := range []string{"default", "gradient", "powerline"} {
		_, err := Resolve(n, nil)
		require.NoError(t, err, n)
	}
}

func TestResolve_Default(t *testing.T) {
	th, err := Resolve("default", nil)
	require.NoError(t, err)
	require.Equal(t, " · ", th.Separator)
	require.Equal(t, [2]rune{'▓', '░'}, th.BarGlyphs)
	require.Equal(t, 10, th.BarWidth)
	require.False(t, th.Gradient)
	require.Equal(t, 32, *th.Style("ok").Ansi)
	require.True(t, th.Style("dim").Dim)
	require.Nil(t, th.Style("nope").Ansi) // unknown role = zero style
}

func TestResolve_GradientInherits(t *testing.T) {
	th, err := Resolve("gradient", nil)
	require.NoError(t, err)
	require.Equal(t, " | ", th.Separator)
	require.Equal(t, 20, th.BarWidth)
	require.True(t, th.Gradient)
	require.True(t, th.Emoji)
	require.Equal(t, "🌿", th.Icons["branch"])
	require.Equal(t, 31, *th.Style("crit").Ansi)  // inherited from default
	require.Equal(t, 35, *th.Style("model").Ansi) // overridden
}

func TestResolve_PowerlineGlyphs(t *testing.T) {
	th, err := Resolve("powerline", nil)
	require.NoError(t, err)
	require.Equal(t, "  ", th.Separator)
	require.Equal(t, "", th.Icons["branch"])
	require.Equal(t, "", th.Icons["dir"])
	require.Equal(t, "\U000F06A9", th.Icons["model"])
}

func TestResolve_UserExtends(t *testing.T) {
	user := map[string]Spec{"mine": {Extends: "gradient", Colors: map[string]string{"accent": "#7aa2f7"}}}
	th, err := Resolve("mine", user)
	require.NoError(t, err)
	require.Equal(t, "mine", th.Name)
	require.Equal(t, 20, th.BarWidth)
	require.NotNil(t, th.Style("accent").FG)
}

func TestResolve_Errors(t *testing.T) {
	_, err := Resolve("nope", nil)
	require.ErrorContains(t, err, "unknown theme")

	cyc := map[string]Spec{"a": {Extends: "b"}, "b": {Extends: "a"}}
	_, err = Resolve("a", cyc)
	require.ErrorContains(t, err, "cycle")

	bad := map[string]Spec{"x": {Colors: map[string]string{"ok": "blurple"}}}
	_, err = Resolve("x", bad)
	require.ErrorContains(t, err, `"ok"`)

	w := 0
	_, err = Resolve("y", map[string]Spec{"y": {BarWidth: &w}})
	require.Error(t, err)
}

func TestParseGlyphs(t *testing.T) {
	g, err := ParseGlyphs("#-")
	require.NoError(t, err)
	require.Equal(t, [2]rune{'#', '-'}, g)
	_, err = ParseGlyphs("#")
	require.Error(t, err)
	_, err = ParseGlyphs("#-+")
	require.Error(t, err)
}

func TestNames(t *testing.T) {
	names := Names(map[string]Spec{"zeta": {}, "alpha": {}, "default": {}})
	require.Equal(t, []string{"default", "gradient", "powerline", "alpha", "zeta"}, names)
}
