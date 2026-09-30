package render

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBar(t *testing.T) {
	f, s := Bar(37, 10, [2]rune{'▓', '░'})
	require.Equal(t, 4, f)
	require.Equal(t, "▓▓▓▓░░░░░░", s)
	_, s = Bar(0, 10, [2]rune{'#', '-'})
	require.Equal(t, "----------", s)
	f, s = Bar(150, 10, [2]rune{'#', '-'})
	require.Equal(t, 10, f)
	require.Equal(t, "##########", s)
	_, s = Bar(-5, 4, [2]rune{'#', '-'})
	require.Equal(t, "----", s)
}

func TestGradientRGB(t *testing.T) {
	require.Equal(t, RGB{0, 200, 80}, GradientRGB(0))
	require.Equal(t, RGB{220, 200, 0}, GradientRGB(0.5))
	require.Equal(t, RGB{220, 40, 20}, GradientRGB(1))
}

func TestPaintBar_GradientOnlyTruecolor(t *testing.T) {
	fill, _ := ParseStyle("green")
	empty, _ := ParseStyle("dim")
	plain := PaintBar(50, 4, [2]rune{'#', '-'}, fill, empty, true, Level256)
	require.Equal(t, "\x1b[32m##\x1b[0m\x1b[2m--\x1b[0m", plain)
	grad := PaintBar(50, 4, [2]rune{'#', '-'}, fill, empty, true, LevelTrue)
	require.Contains(t, grad, "38;2;")
	require.Equal(t, 4, Width(grad))
}
