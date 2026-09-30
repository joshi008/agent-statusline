package config

import (
	"fmt"
	"strings"

	"github.com/joshi008/agent-statusline/internal/theme"
)

// Validate reports every problem at once. known reports whether a built-in segment id exists
// (passed in to avoid an import cycle with the segments package).
func Validate(c *Config, known func(string) bool) []error {
	var errs []error
	add := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }

	if c.Version != 1 {
		add("version: must be 1, got %d", c.Version)
	}
	switch c.Color {
	case "", "auto", "none", "basic", "256", "truecolor":
	default:
		add("color: %q is not one of auto, none, basic, 256, truecolor", c.Color)
	}
	if c.Style.Thresholds.Warn >= c.Style.Thresholds.Crit {
		add("style.thresholds: warn (%v) must be below crit (%v)", c.Style.Thresholds.Warn, c.Style.Thresholds.Crit)
	}
	if g := c.Style.Bar.Glyphs; g != nil {
		if _, err := theme.ParseGlyphs(*g); err != nil {
			add("style.bar.glyphs: %v", err)
		}
	}
	if _, err := theme.Resolve(c.Theme, c.Themes); err != nil {
		add("theme: %v", err)
	}

	seen := map[string]bool{}
	for i, cu := range c.Custom {
		switch {
		case cu.ID == "":
			add("custom[%d]: id is required", i)
		case seen[cu.ID]:
			add("custom: duplicate id %q", cu.ID)
		}
		seen[cu.ID] = true
		if (cu.Run == "") == (cu.Text == "") {
			add("custom %q: set exactly one of run or text", cu.ID)
		}
	}

	checkLines := func(where string, lines []Line) {
		for li, line := range lines {
			for _, ref := range line {
				kind, name, isCustom := strings.Cut(ref.ID, ":")
				if isCustom && (kind == "cmd" || kind == "text") {
					cu, ok := c.FindCustom(name)
					switch {
					case !ok:
						add("%s line %d: unknown custom segment %q", where, li+1, name)
					case kind == "cmd" && cu.Run == "":
						add("%s line %d: %q is a text segment; reference it as text:%s", where, li+1, name, name)
					case kind == "text" && cu.Text == "":
						add("%s line %d: %q is a cmd segment; reference it as cmd:%s", where, li+1, name, name)
					}
					continue
				}
				if !known(ref.ID) {
					add("%s line %d: unknown segment %q", where, li+1, ref.ID)
				}
			}
		}
	}
	checkLines("claude.lines", c.Claude.Lines)
	checkLines("cursor.lines", c.Cursor.Lines)

	if c.Claude.RefreshInterval < 0 {
		add("claude.refresh_interval: must be >= 0 seconds (0 = events only)")
	}
	if c.Cursor.UpdateIntervalMs != 0 && c.Cursor.UpdateIntervalMs < 300 {
		add("cursor.update_interval_ms: minimum is 300 (Cursor clamps lower values)")
	}
	if c.Cursor.TimeoutMs != 0 && c.Cursor.TimeoutMs < 50 {
		add("cursor.timeout_ms: minimum is 50")
	}
	return errs
}
