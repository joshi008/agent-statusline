package render

import (
	"math"
	"strings"
)

// Bar returns the filled-block count and the plain bar string for pct (clamped 0..100).
func Bar(pct float64, width int, glyphs [2]rune) (int, string) {
	if width <= 0 {
		return 0, ""
	}
	pct = math.Max(0, math.Min(100, pct))
	filled := int(math.Round(pct / 100 * float64(width)))
	return filled, strings.Repeat(string(glyphs[0]), filled) + strings.Repeat(string(glyphs[1]), width-filled)
}

// GradientRGB interpolates green(0,200,80) → yellow(220,200,0) → red(220,40,20) for pos 0..1.
func GradientRGB(pos float64) RGB {
	pos = math.Max(0, math.Min(1, pos))
	if pos <= 0.5 {
		t := pos / 0.5
		return RGB{uint8(220 * t), 200, uint8(80 - 80*t)}
	}
	t := (pos - 0.5) / 0.5
	return RGB{220, uint8(200 - 160*t), uint8(20 * t)}
}

// PaintBar colours the filled part with fill (or a per-block gradient at truecolor) and the rest
// with empty. Output display width always equals width.
func PaintBar(pct float64, width int, glyphs [2]rune, fill, empty Style, gradient bool, lvl Level) string {
	filled, _ := Bar(pct, width, glyphs)
	var b strings.Builder
	if gradient && lvl == LevelTrue {
		for i := 0; i < filled; i++ {
			pos := 0.0
			if width > 1 {
				pos = float64(i) / float64(width-1)
			}
			c := GradientRGB(pos)
			b.WriteString(Style{FG: &c}.Paint(string(glyphs[0]), lvl))
		}
	} else {
		b.WriteString(fill.Paint(strings.Repeat(string(glyphs[0]), filled), lvl))
	}
	b.WriteString(empty.Paint(strings.Repeat(string(glyphs[1]), width-filled), lvl))
	return b.String()
}
