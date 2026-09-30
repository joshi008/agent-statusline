package segments

import "strings"

func init() {
	register(Def{ID: "session", Desc: "Session name or title", Claude: true, Cursor: true, Codex: "thread-title",
		Render: func(e *Env) []Cell { return badge(true, e.Snap.SessionName, "session") }})
	register(Def{ID: "model", Desc: "Model name", Claude: true, Cursor: true, Codex: "model", Render: renderModel})
	register(Def{ID: "effort", Desc: "Reasoning effort (low…max)", Claude: true, Cursor: true, Codex: "reasoning",
		Render: func(e *Env) []Cell {
			if e.Snap.Effort == "" {
				return nil
			}
			return badge(true, strings.ReplaceAll(e.Opts.String("format", "{level}"), "{level}", e.Snap.Effort), "effort")
		}})
	register(Def{ID: "thinking", Desc: "Extended thinking badge", Claude: true,
		Render: func(e *Env) []Cell {
			return badge(e.Snap.Thinking != nil && *e.Snap.Thinking, e.Opts.String("label", "think"), "dim")
		}})
	register(Def{ID: "fast", Desc: "Fast mode badge", Claude: true, Codex: "fast-mode",
		Render: func(e *Env) []Cell {
			return badge(e.Snap.FastMode != nil && *e.Snap.FastMode, e.Opts.String("label", "fast"), "warn")
		}})
	register(Def{ID: "max_mode", Desc: "Cursor Max Mode badge", Cursor: true,
		Render: func(e *Env) []Cell { return badge(e.Snap.Model.MaxMode, e.Opts.String("label", "max"), "accent") }})
	register(Def{ID: "autorun", Desc: "Cursor autorun / manual approvals", Cursor: true,
		Render: func(e *Env) []Cell {
			if e.Snap.Autorun == nil {
				return nil
			}
			if *e.Snap.Autorun {
				return badge(true, e.Opts.String("label_on", "auto"), "warn")
			}
			return badge(true, e.Opts.String("label_off", "manual"), "dim")
		}})
	register(Def{ID: "agent", Desc: "Agent name (--agent)", Claude: true,
		Render: func(e *Env) []Cell {
			if e.Snap.Agent == "" {
				return nil
			}
			return []Cell{{Text: "agent:", Role: "label"}, {Text: e.Snap.Agent, Role: "plain"}}
		}})
	register(Def{ID: "vim", Desc: "Vim mode", Claude: true, Cursor: true,
		Render: func(e *Env) []Cell { return badge(true, e.Snap.Vim, "accent") }})
	register(Def{ID: "version", Desc: "Harness version", Claude: true, Cursor: true, Codex: "codex-version",
		Render: func(e *Env) []Cell {
			if e.Snap.Version == "" {
				return nil
			}
			return badge(true, "v"+e.Snap.Version, "dim")
		}})
	register(Def{ID: "time", Desc: "Wall clock", Claude: true, Cursor: true,
		Render: func(e *Env) []Cell { return badge(true, e.Snap.Now.Format(e.Opts.String("format", "15:04")), "dim") }})
}

func renderModel(e *Env) []Cell {
	name := e.Snap.Model.DisplayName
	if name == "" {
		name = e.Snap.Model.ID
	}
	if name == "" {
		return nil
	}
	if e.Opts.Bool("show_id", false) && e.Snap.Model.ID != "" && e.Snap.Model.ID != name {
		name += " (" + e.Snap.Model.ID + ")"
	}
	return []Cell{{Text: name, Role: "model"}}
}
