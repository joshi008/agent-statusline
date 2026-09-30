package segments

import (
	"fmt"
	"math"
	"strings"

	"github.com/joshi008/agent-statusline/internal/render"
)

func init() {
	register(Def{ID: "context", Desc: "Context window used (bar / % / tokens)", Claude: true, Cursor: true, Codex: "context-used", Render: renderContext})
	register(Def{ID: "tokens", Desc: "Input / output tokens", Claude: true, Cursor: true, Codex: "used-tokens",
		Render: func(e *Env) []Cell {
			c := e.Snap.Context
			if c == nil || (c.TotalInput == 0 && c.TotalOutput == 0) {
				return nil
			}
			return badge(true, "in "+Humanize(c.TotalInput)+" out "+Humanize(c.TotalOutput), "dim")
		}})
	register(Def{ID: "cache", Desc: "Prompt cache warm / hit ratio", Claude: true,
		Render: func(e *Env) []Cell {
			c := e.Snap.Cache
			if c == nil {
				return nil
			}
			if !c.Warm {
				return badge(true, "cache cold", "warn")
			}
			txt := "cache"
			if c.HitRatio != nil {
				txt += " " + pctText(*c.HitRatio*100)
			}
			return badge(true, txt, "ok")
		}})
	register(Def{ID: "cost", Desc: "Session cost (USD)", Claude: true, Codex: "estimated-thread-cost",
		Render: func(e *Env) []Cell {
			if e.Snap.Cost == nil {
				return nil
			}
			return badge(true, fmt.Sprintf(e.Opts.String("format", "$%.2f"), e.Snap.Cost.USD), "cost")
		}})
	register(Def{ID: "duration", Desc: "Session wall-clock time", Claude: true,
		Render: func(e *Env) []Cell {
			c := e.Snap.Cost
			if c == nil || c.DurationMs <= 0 {
				return nil
			}
			txt := render.Duration(c.DurationMs)
			if e.Opts.Bool("api", false) && c.APIDurationMs > 0 {
				txt += " (api " + render.Duration(c.APIDurationMs) + ")"
			}
			return badge(true, txt, "dim")
		}})
	register(Def{ID: "lines", Desc: "Lines added / removed", Claude: true,
		Render: func(e *Env) []Cell {
			c := e.Snap.Cost
			if c == nil || (c.LinesAdded == 0 && c.LinesRemoved == 0) {
				return nil
			}
			return []Cell{{Text: fmt.Sprintf("+%d", c.LinesAdded), Role: "ok"}, {Text: fmt.Sprintf(" -%d", c.LinesRemoved), Role: "crit"}}
		}})
	register(Def{ID: "limit_5h", Desc: "5-hour rate limit + reset countdown", Claude: true, Codex: "five-hour-limit", Render: limitRender("5h", "5h", "bar+pct")})
	register(Def{ID: "limit_7d", Desc: "7-day rate limit + reset countdown", Claude: true, Codex: "weekly-limit", Render: limitRender("7d", "7d", "bar+pct")})
	register(Def{ID: "spend", Desc: "Spend limit (apps gateway)", Claude: true, Render: limitRender("spend", "spend", "pct")})
}

func pctText(p float64) string { return fmt.Sprintf("%d%%", int(math.Round(p))) }

func labelCells(e *Env, def string) []Cell {
	if l := e.Opts.String("label", def); l != "" {
		return []Cell{{Text: l + " ", Role: "label"}}
	}
	return nil
}

// meterCells renders "bar", "pct", "bar+pct" (and the bar half of "bar+tokens").
func meterCells(e *Env, pct float64, role, style string) []Cell {
	var cells []Cell
	hasBar := strings.HasPrefix(style, "bar")
	if hasBar {
		w := e.Opts.Int("bar_width", e.Theme.BarWidth)
		cells = append(cells, Cell{Raw: true, Text: render.PaintBar(pct, w, e.Theme.BarGlyphs,
			e.Theme.Style(role), e.Theme.Style("bar_empty"), e.Theme.Gradient, e.Level)})
	}
	if strings.HasSuffix(style, "pct") {
		sp := ""
		if hasBar {
			sp = " "
		}
		cells = append(cells, Cell{Text: sp + pctText(pct), Role: role})
	}
	return cells
}

func emojiTier(pct float64) string {
	switch {
	case pct < 20:
		return "🟢"
	case pct < 70:
		return "⚡"
	case pct < 90:
		return "🔥"
	}
	return "🚨"
}

func renderContext(e *Env) []Cell {
	c := e.Snap.Context
	if c == nil || c.UsedPct == nil {
		return nil
	}
	pct := *c.UsedPct
	role := LevelRole(pct, e.Warn, e.Crit)
	style := e.Opts.String("style", "bar+pct")
	cells := labelCells(e, "ctx")
	if e.Theme.Emoji {
		cells = append(cells, Cell{Text: emojiTier(pct) + " "})
	}
	if strings.HasSuffix(style, "tokens") {
		if strings.HasPrefix(style, "bar") {
			cells = append(cells, meterCells(e, pct, role, "bar")...)
			cells = append(cells, Cell{Text: " "})
		}
		if c.Size != nil && *c.Size > 0 {
			cells = append(cells, Cell{Text: Humanize(c.TotalInput) + "/" + Humanize(*c.Size), Role: role})
		}
		return cells
	}
	return append(cells, meterCells(e, pct, role, style)...)
}

func limitRender(key, label, defStyle string) func(e *Env) []Cell {
	return func(e *Env) []Cell {
		l, ok := e.Snap.Limits[key]
		if !ok {
			return nil
		}
		role := LevelRole(l.UsedPct, e.Warn, e.Crit)
		cells := append(labelCells(e, label), meterCells(e, l.UsedPct, role, e.Opts.String("style", defStyle))...)
		if e.Countdown && e.Opts.Bool("countdown", true) && l.ResetsAt != nil {
			if cd := render.Countdown(*l.ResetsAt, e.Snap.Now); cd != "" {
				cells = append(cells, Cell{Text: " " + cd, Role: "dim"})
			}
		}
		return cells
	}
}
