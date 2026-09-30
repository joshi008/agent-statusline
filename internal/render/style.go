package render

import (
	"fmt"
	"strconv"
	"strings"
)

type RGB struct{ R, G, B uint8 }

// Style is a parsed "tokens" string such as "#7aa2f7 bold" or "cyan dim" or "208".
type Style struct {
	FG        *RGB
	Ansi      *int // 30-37, 90-97 (basic) or 0-255 (256 palette) — see ansiIs256
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
}

var named = map[string]int{
	"black": 30, "red": 31, "green": 32, "yellow": 33, "blue": 34, "magenta": 35, "cyan": 36, "white": 37,
	"bright-black": 90, "bright-red": 91, "bright-green": 92, "bright-yellow": 93, "bright-blue": 94,
	"bright-magenta": 95, "bright-cyan": 96, "bright-white": 97,
}

// ParseStyle parses whitespace-separated tokens. Empty string is a valid zero Style.
func ParseStyle(s string) (Style, error) {
	var st Style
	for _, tok := range strings.Fields(s) {
		switch {
		case tok == "bold":
			st.Bold = true
		case tok == "dim":
			st.Dim = true
		case tok == "italic":
			st.Italic = true
		case tok == "underline":
			st.Underline = true
		case strings.HasPrefix(tok, "#"):
			if len(tok) != 7 {
				return st, fmt.Errorf("style: bad hex colour %q", tok)
			}
			v, err := strconv.ParseUint(tok[1:], 16, 32)
			if err != nil {
				return st, fmt.Errorf("style: bad hex colour %q", tok)
			}
			st.FG = &RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}
			st.Ansi = nil
		default:
			if n, ok := named[tok]; ok {
				st.Ansi = &n
				st.FG = nil
				continue
			}
			n, err := strconv.Atoi(tok)
			if err != nil || n < 0 || n > 255 {
				return st, fmt.Errorf("style: unknown token %q", tok)
			}
			st.Ansi = &n
			st.FG = nil
		}
	}
	return st, nil
}

// Merge returns base overridden by every non-zero field of over.
func (s Style) Merge(over Style) Style {
	out := s
	if over.FG != nil {
		out.FG, out.Ansi = over.FG, nil
	}
	if over.Ansi != nil {
		out.Ansi, out.FG = over.Ansi, nil
	}
	out.Bold = out.Bold || over.Bold
	out.Dim = out.Dim || over.Dim
	out.Italic = out.Italic || over.Italic
	out.Underline = out.Underline || over.Underline
	return out
}

func (s Style) isZero() bool {
	return s.FG == nil && s.Ansi == nil && !s.Bold && !s.Dim && !s.Italic && !s.Underline
}

// SGR returns the escape prefix for this style at the given level, "" when nothing applies.
func (s Style) SGR(lvl Level) string {
	if lvl == LevelNone || s.isZero() {
		return ""
	}
	var codes []string
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	if c := s.colorCode(lvl); c != "" {
		codes = append(codes, c)
	}
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// Paint wraps text in the style's SGR and a reset. Empty text stays empty.
func (s Style) Paint(text string, lvl Level) string {
	if text == "" {
		return ""
	}
	p := s.SGR(lvl)
	if p == "" {
		return text
	}
	return p + text + "\x1b[0m"
}

func (s Style) colorCode(lvl Level) string {
	switch {
	case s.FG != nil:
		switch lvl {
		case LevelTrue:
			return fmt.Sprintf("38;2;%d;%d;%d", s.FG.R, s.FG.G, s.FG.B)
		case Level256:
			return fmt.Sprintf("38;5;%d", to256(*s.FG))
		default:
			return strconv.Itoa(toBasic(*s.FG))
		}
	case s.Ansi != nil:
		n := *s.Ansi
		if (n >= 30 && n <= 37) || (n >= 90 && n <= 97) {
			return strconv.Itoa(n)
		}
		if lvl == LevelBasic {
			return strconv.Itoa(30 + n%8)
		}
		return fmt.Sprintf("38;5;%d", n)
	}
	return ""
}

func to256(c RGB) int {
	q := func(v uint8) int { return int((float64(v)/255)*5 + 0.5) }
	return 16 + 36*q(c.R) + 6*q(c.G) + q(c.B)
}

// toBasic maps to the nearest of the 8 basic colours by RGB distance.
func toBasic(c RGB) int {
	basics := []RGB{{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0}, {0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229}}
	best, bestD := 0, 1<<30
	for i, b := range basics {
		d := sq(int(c.R)-int(b.R)) + sq(int(c.G)-int(b.G)) + sq(int(c.B)-int(b.B))
		if d < bestD {
			best, bestD = i, d
		}
	}
	return 30 + best
}

func sq(x int) int { return x * x }
