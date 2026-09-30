package config

import "fmt"

// PresetNames lists layout presets in wizard order.
var PresetNames = []string{"two-line", "one-line", "minimal"}

var presets = map[string]struct{ claude, cursor [][]string }{
	"two-line": {
		claude: [][]string{
			{"session", "model", "effort", "thinking", "fast", "dir", "branch", "git_status", "pr", "worktree", "agent"},
			{"context", "cost", "limit_5h", "limit_7d", "spend", "cache"},
		},
		cursor: [][]string{
			{"model", "max_mode", "autorun", "dir", "branch", "git_status", "pr", "worktree"},
			{"context", "vim"},
		},
	},
	"one-line": {
		claude: [][]string{{"session", "model", "effort", "worktree", "cost", "context", "limit_5h", "limit_7d", "agent"}},
		cursor: [][]string{{"model", "max_mode", "autorun", "dir", "branch", "pr", "context"}},
	},
	"minimal": {
		claude: [][]string{{"model", "context", "limit_5h"}},
		cursor: [][]string{{"model", "context"}},
	},
}

// ApplyPreset replaces the Claude and Cursor lines with a preset.
func ApplyPreset(c *Config, name string) error {
	p, ok := presets[name]
	if !ok {
		return fmt.Errorf("unknown preset %q (want one of %v)", name, PresetNames)
	}
	c.Claude.Lines = linesOf(p.claude)
	c.Cursor.Lines = linesOf(p.cursor)
	return nil
}

func linesOf(rows [][]string) []Line {
	out := make([]Line, 0, len(rows))
	for _, row := range rows {
		l := make(Line, 0, len(row))
		for _, id := range row {
			l = append(l, SegRef{ID: id})
		}
		out = append(out, l)
	}
	return out
}
