// Package segments holds every status line segment. A segment turns a Snapshot into styled
// cells, or returns nil to be skipped (the layout then emits no separator for it).
package segments

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/theme"
)

// Cell is one styled run of text. Raw cells are already painted (bars, hyperlinks).
type Cell struct {
	Text string
	Role string
	Raw  bool
}

// Env is everything a segment may read.
type Env struct {
	Snap      *model.Snapshot
	Theme     *theme.Theme
	Level     render.Level
	Warn      float64
	Crit      float64
	Countdown bool
	Opts      Options
	Getenv    func(string) string
}

// Need tells the engine which enrichment a segment depends on.
type Need int

const (
	NeedGit Need = 1 << iota
	NeedRepo
	NeedPR
	NeedCustom
)

// Def describes a segment. Codex is the Codex item id it maps to ("" when none).
type Def struct {
	ID     string
	Desc   string
	Claude bool
	Cursor bool
	Codex  string
	Needs  Need
	Render func(e *Env) []Cell
}

// Available reports whether the segment can show data on a harness.
func (d *Def) Available(harness string) bool {
	switch harness {
	case "claude":
		return d.Claude
	case "cursor":
		return d.Cursor
	case "codex":
		return d.Codex != ""
	}
	return false
}

// CatalogOrder is the display order used by All(), the picker and docs.
var CatalogOrder = []string{
	"session", "model", "effort", "thinking", "fast", "max_mode", "autorun",
	"dir", "repo", "branch", "git_status", "worktree", "pr", "agent", "vim", "version",
	"context", "tokens", "cache", "cost", "duration", "lines", "limit_5h", "limit_7d", "spend", "time",
}

var registry = map[string]*Def{}

func register(d Def) {
	if _, dup := registry[d.ID]; dup {
		panic("segments: duplicate id " + d.ID)
	}
	registry[d.ID] = &d
}

// Lookup finds a built-in segment, or builds a custom one for "cmd:<id>" / "text:<id>".
func Lookup(id string) (*Def, bool) {
	if kind, name, ok := strings.Cut(id, ":"); ok {
		if (kind == "cmd" || kind == "text") && name != "" {
			return customDef(id, name), true
		}
		return nil, false
	}
	d, ok := registry[id]
	return d, ok
}

// Known reports whether id resolves to a segment.
func Known(id string) bool { _, ok := Lookup(id); return ok }

// All returns registered built-in segments in catalogue order.
func All() []*Def {
	out := make([]*Def, 0, len(CatalogOrder))
	for _, id := range CatalogOrder {
		if d, ok := registry[id]; ok {
			out = append(out, d)
		}
	}
	return out
}

// Options are per-segment settings from config (`segments:` merged with inline line options).
type Options map[string]any

func (o Options) String(k, def string) string {
	if v, ok := o[k].(string); ok {
		return v
	}
	return def
}

func (o Options) Int(k string, def int) int {
	switch v := o[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return def
}

func (o Options) Bool(k string, def bool) bool {
	if v, ok := o[k].(bool); ok {
		return v
	}
	return def
}

// MergeOptions returns base overridden by over; neither input is modified.
func MergeOptions(base, over map[string]any) Options {
	out := Options{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// LevelRole maps a used-percentage to a colour role.
func LevelRole(pct, warn, crit float64) string {
	switch {
	case pct < warn:
		return "ok"
	case pct < crit:
		return "warn"
	}
	return "crit"
}

// Humanize formats token counts: 950, 4.1k, 372k, 1M.
func Humanize(n int64) string {
	trim := func(s string) string { return strings.TrimSuffix(s, ".0") }
	switch {
	case n < 1000:
		return strconv.FormatInt(n, 10)
	case n < 1_000_000:
		return trim(fmt.Sprintf("%.1f", float64(n)/1000)) + "k"
	}
	return trim(fmt.Sprintf("%.1f", float64(n)/1_000_000)) + "M"
}

// badge renders a single-cell segment when cond holds.
func badge(cond bool, text, role string) []Cell {
	if !cond || text == "" {
		return nil
	}
	return []Cell{{Text: text, Role: role}}
}
