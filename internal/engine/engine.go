package engine

import (
	"context"
	"strings"
	"time"

	"github.com/joshi008/agent-statusline/internal/cache"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/enrich"
	"github.com/joshi008/agent-statusline/internal/harness"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/segments"
	"github.com/joshi008/agent-statusline/internal/theme"
)

type Input struct {
	Payload      []byte
	Harness      string // claude | cursor | auto
	Config       *config.Config
	Getenv       func(string) string
	NoColor      bool
	Now          time.Time // zero = wall clock
	Width        int       // > 0 overrides COLUMNS / render_width_chars
	Cache        *cache.Cache
	DisableCache bool
	Run          enrich.Runner
	Logf         func(format string, args ...any)
}

// Render produces the status line. It never panics and never returns an empty string.
func Render(in Input) (out string) {
	logf := in.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	getenv := in.Getenv
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	defer func() {
		if r := recover(); r != nil {
			logf("render panic: %v", r)
			out = "agent-statusline\n"
		}
	}()

	cfg := in.Config
	if cfg == nil {
		cfg = config.Default()
	}
	snap, err := harness.Parse(in.Payload, in.Harness, getenv)
	if err != nil {
		logf("payload: %v", err)
		h := model.HarnessClaude
		if in.Harness == "cursor" {
			h = model.HarnessCursor
		}
		snap = &model.Snapshot{Harness: h, Model: model.ModelInfo{DisplayName: "?"}, Limits: map[string]model.Limit{}, Now: time.Now()}
	}
	if !in.Now.IsZero() {
		snap.Now = in.Now
	}
	if in.Width > 0 {
		snap.Width = in.Width
	}

	th := ResolveTheme(cfg, logf)
	lvl := ColorLevel(cfg.Color, in.NoColor, getenv)
	lines := cfg.LinesFor(snap.Harness)

	req := enrich.Request{Run: in.Run, Logf: logf}
	req.Needs, req.Custom = collectNeeds(cfg, lines)
	if req.Needs != 0 || len(req.Custom) > 0 {
		if !in.DisableCache {
			req.Cache = in.Cache
			if req.Cache == nil {
				req.Cache = cache.New(cache.DefaultDir(getenv, getenv("HOME")), snap.SessionID)
			}
		}
		enrich.Enrich(context.Background(), snap, req)
	}

	warn, crit := cfg.Style.Thresholds.Warn, cfg.Style.Thresholds.Crit
	if crit <= 0 || warn >= crit {
		warn, crit = 50, 80
	}
	base := segments.Env{Snap: snap, Theme: th, Level: lvl, Warn: warn, Crit: crit,
		Countdown: cfg.Style.Countdown == nil || *cfg.Style.Countdown, Getenv: getenv}

	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		if row := FitLine(renderParts(line, base, cfg, logf), th, lvl, snap.Width); row != "" {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows, fallbackText(snap))
	}
	return strings.Join(rows, "\n") + "\n"
}

func renderParts(line config.Line, base segments.Env, cfg *config.Config, logf func(string, ...any)) []Part {
	parts := make([]Part, 0, len(line))
	for _, ref := range line {
		d, ok := segments.Lookup(ref.ID)
		if !ok {
			logf("unknown segment %q", ref.ID)
			continue
		}
		env := base
		env.Opts = segments.MergeOptions(cfg.SegmentOpts(ref.ID), ref.Opts)
		parts = append(parts, Part{ID: ref.ID, Text: paint(ref.ID, safeRender(d, &env, logf), base.Theme, base.Level)})
	}
	return parts
}

func safeRender(d *segments.Def, e *segments.Env, logf func(string, ...any)) (cells []segments.Cell) {
	defer func() {
		if r := recover(); r != nil {
			logf("segment %s panic: %v", d.ID, r)
			cells = nil
		}
	}()
	return d.Render(e)
}

func paint(id string, cells []segments.Cell, th *theme.Theme, lvl render.Level) string {
	if len(cells) == 0 {
		return ""
	}
	var b strings.Builder
	if icon := th.Icons[id]; icon != "" {
		b.WriteString(th.Style(cells[0].Role).Paint(icon, lvl))
		b.WriteString(" ")
	}
	for _, c := range cells {
		if c.Raw {
			b.WriteString(c.Text)
		} else {
			b.WriteString(th.Style(c.Role).Paint(c.Text, lvl))
		}
	}
	return b.String()
}

func collectNeeds(cfg *config.Config, lines []config.Line) (segments.Need, []config.Custom) {
	var need segments.Need
	var customs []config.Custom
	seen := map[string]bool{}
	for _, line := range lines {
		for _, ref := range line {
			d, ok := segments.Lookup(ref.ID)
			if !ok {
				continue
			}
			need |= d.Needs
			if _, name, isCustom := strings.Cut(ref.ID, ":"); isCustom && !seen[name] {
				if c, ok := cfg.FindCustom(name); ok {
					customs = append(customs, c)
					seen[name] = true
				}
			}
		}
	}
	return need &^ segments.NeedCustom, customs
}

func fallbackText(s *model.Snapshot) string {
	if s.Model.DisplayName != "" {
		return s.Model.DisplayName
	}
	if s.Model.ID != "" {
		return s.Model.ID
	}
	return "agent-statusline"
}

// ResolveTheme loads the configured theme (default on error) and applies `style:` overrides.
func ResolveTheme(cfg *config.Config, logf func(string, ...any)) *theme.Theme {
	th, err := theme.Resolve(cfg.Theme, cfg.Themes)
	if err != nil {
		logf("theme: %v (using default)", err)
		th, _ = theme.Resolve("default", nil)
	}
	s := cfg.Style
	if s.Separator != nil {
		th.Separator = *s.Separator
	}
	if s.Bar.Width != nil && *s.Bar.Width > 0 && *s.Bar.Width <= 50 {
		th.BarWidth = *s.Bar.Width
	}
	if s.Bar.Glyphs != nil {
		if g, err := theme.ParseGlyphs(*s.Bar.Glyphs); err == nil {
			th.BarGlyphs = g
		} else {
			logf("style.bar.glyphs: %v", err)
		}
	}
	if s.Emoji != nil {
		th.Emoji = *s.Emoji
	}
	return th
}

// ColorLevel resolves `color:` (auto|none|basic|256|truecolor) with --no-color and NO_COLOR.
func ColorLevel(color string, noColor bool, getenv func(string) string) render.Level {
	if noColor || getenv("NO_COLOR") != "" {
		return render.LevelNone
	}
	if l, ok := render.ParseLevel(color); ok {
		return l
	}
	return render.DetectLevel(getenv)
}
