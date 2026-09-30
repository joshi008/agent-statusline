// Package engine turns a payload plus config into the final status line text.
package engine

import (
	"strings"

	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/theme"
)

// Part is one rendered segment; Text is already painted and may be empty (skipped).
type Part struct {
	ID   string
	Text string
}

// DropOrder lists segment ids from first-dropped to last-dropped when a line is too wide.
var DropOrder = []string{
	"text", "cmd", "time", "version", "vim", "agent", "worktree", "repo", "git_status", "lines",
	"duration", "cache", "tokens", "spend", "pr", "dir", "thinking", "fast", "max_mode", "autorun",
	"session", "effort", "branch", "limit_7d", "limit_5h", "cost", "context", "model",
}

func dropRank(id string) int {
	if kind, _, ok := strings.Cut(id, ":"); ok {
		id = kind
	}
	for i, d := range DropOrder {
		if d == id {
			return i
		}
	}
	return -1 // unknown ids go first
}

// JoinLine joins non-empty parts with the theme separator, wrapping in caps when set.
func JoinLine(parts []Part, th *theme.Theme, lvl render.Level) string {
	sepStyle := th.Style("sep")
	sep := sepStyle.Paint(th.Separator, lvl)
	var b strings.Builder
	n := 0
	for _, p := range parts {
		if render.Width(p.Text) == 0 {
			continue
		}
		if n > 0 {
			b.WriteString(sep)
		}
		b.WriteString(p.Text)
		n++
	}
	if n == 0 {
		return ""
	}
	if th.Caps[0] == "" && th.Caps[1] == "" {
		return b.String()
	}
	return sepStyle.Paint(th.Caps[0], lvl) + b.String() + sepStyle.Paint(th.Caps[1], lvl)
}

// FitLine drops low-priority parts until the line fits width, then truncates the survivor.
// width <= 0 disables fitting.
func FitLine(parts []Part, th *theme.Theme, lvl render.Level, width int) string {
	visible := make([]Part, 0, len(parts))
	for _, p := range parts {
		if render.Width(p.Text) > 0 {
			visible = append(visible, p)
		}
	}
	line := JoinLine(visible, th, lvl)
	if width <= 0 {
		return line
	}
	for render.Width(line) > width && len(visible) > 1 {
		victim := 0
		for i, p := range visible {
			if dropRank(p.ID) <= dropRank(visible[victim].ID) {
				victim = i // lowest rank wins; ties go to the rightmost
			}
		}
		visible = append(visible[:victim], visible[victim+1:]...)
		line = JoinLine(visible, th, lvl)
	}
	if render.Width(line) > width {
		line = render.Truncate(line, width)
	}
	return line
}
