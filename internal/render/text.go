package render

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\]8;;[^\x07]*\x07`)

// Strip removes SGR and OSC 8 sequences.
func Strip(s string) string { return ansiRe.ReplaceAllString(s, "") }

// Width is the display width of s ignoring escape sequences.
func Width(s string) int { return runewidth.StringWidth(Strip(s)) }

// Truncate cuts visible text to w columns (appending "…" when cut) while keeping every escape
// sequence intact and re-emitting them so the string stays balanced.
func Truncate(s string, w int) string {
	if Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	budget := w - 1 // room for the ellipsis
	var out strings.Builder
	rest := s
	for rest != "" {
		if loc := ansiRe.FindStringIndex(rest); loc != nil && loc[0] == 0 {
			out.WriteString(rest[:loc[1]])
			rest = rest[loc[1]:]
			continue
		}
		r := []rune(rest)[0]
		rw := runewidth.RuneWidth(r)
		if budget-rw < 0 {
			break
		}
		budget -= rw
		out.WriteRune(r)
		rest = rest[len(string(r)):]
	}
	out.WriteString("…")
	// re-append any escape sequences remaining after the cut so resets are not lost
	for _, m := range ansiRe.FindAllString(rest, -1) {
		out.WriteString(m)
	}
	return out.String()
}

// Countdown formats resetsAt - now as ~2d3h, ~1h29m, ~12m or <1m; "" when not in the future.
func Countdown(resetsAt int64, now time.Time) string {
	d := resetsAt - now.Unix()
	if d <= 0 {
		return ""
	}
	days, hrs, mins := d/86400, (d%86400)/3600, (d%3600)/60
	switch {
	case days > 0:
		return fmt.Sprintf("~%dd%dh", days, hrs)
	case hrs > 0:
		return fmt.Sprintf("~%dh%dm", hrs, mins)
	case mins > 0:
		return fmt.Sprintf("~%dm", mins)
	}
	return "<1m"
}

// Duration formats milliseconds as 45s, 12m5s, 1h0m.
func Duration(ms int64) string {
	sec := ms / 1000
	h, m, s := sec/3600, (sec%3600)/60, sec%60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
