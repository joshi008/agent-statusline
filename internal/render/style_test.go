package render

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStyle(t *testing.T) {
	s, err := ParseStyle("#7aa2f7 bold")
	require.NoError(t, err)
	require.Equal(t, RGB{0x7a, 0xa2, 0xf7}, *s.FG)
	require.True(t, s.Bold)

	s, err = ParseStyle("cyan dim")
	require.NoError(t, err)
	require.Equal(t, 36, *s.Ansi)
	require.True(t, s.Dim)

	s, err = ParseStyle("208")
	require.NoError(t, err)
	require.Equal(t, 208, *s.Ansi)

	_, err = ParseStyle("#zz")
	require.Error(t, err)
	_, err = ParseStyle("blurple")
	require.Error(t, err)
}

func TestPaint_Levels(t *testing.T) {
	s, _ := ParseStyle("#ff8800 bold")
	require.Equal(t, "x", s.Paint("x", LevelNone))
	require.Equal(t, "\x1b[1;38;2;255;136;0mx\x1b[0m", s.Paint("x", LevelTrue))
	require.Equal(t, "\x1b[1;38;5;214mx\x1b[0m", s.Paint("x", Level256)) // 16+36*5+6*3+0
	require.Equal(t, "\x1b[1;33mx\x1b[0m", s.Paint("x", LevelBasic))     // nearest basic: yellow
	d, _ := ParseStyle("dim")
	require.Equal(t, "\x1b[2mx\x1b[0m", d.Paint("x", LevelBasic))
	require.Equal(t, "x", Style{}.Paint("x", LevelTrue))
}

func TestMerge(t *testing.T) {
	base, _ := ParseStyle("red bold")
	over, _ := ParseStyle("#00ff00")
	m := base.Merge(over)
	require.Nil(t, m.Ansi)
	require.Equal(t, RGB{0, 255, 0}, *m.FG)
	require.True(t, m.Bold)
}
