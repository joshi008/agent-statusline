package segments

// customDef builds the segment for "cmd:<name>" or "text:<name>". The engine resolves the
// value (running the command through the cache, or copying the literal) into Snap.Custom.
func customDef(id, name string) *Def {
	return &Def{ID: id, Desc: "Custom segment", Claude: true, Cursor: true, Needs: NeedCustom,
		Render: func(e *Env) []Cell {
			v := e.Snap.Custom[name]
			if v == "" {
				return nil
			}
			return []Cell{{Text: v, Role: e.Opts.String("role", "plain")}}
		}}
}
