package segments

import (
	"fmt"
	"strings"

	"github.com/joshi008/agent-statusline/internal/render"
)

func init() {
	register(Def{ID: "dir", Desc: "Working directory (trailing components)", Claude: true, Cursor: true, Codex: "current-dir", Render: renderDir})
	register(Def{ID: "repo", Desc: "Repository owner/name", Claude: true, Cursor: true, Codex: "project-name", Needs: NeedRepo,
		Render: func(e *Env) []Cell {
			r := e.Snap.Repo
			if r == nil || r.Name == "" {
				return nil
			}
			if e.Opts.Bool("name_only", false) || r.Owner == "" {
				return badge(true, r.Name, "repo")
			}
			return badge(true, r.Owner+"/"+r.Name, "repo")
		}})
	register(Def{ID: "branch", Desc: "Git branch", Claude: true, Cursor: true, Codex: "git-branch", Needs: NeedGit,
		Render: func(e *Env) []Cell {
			g := e.Snap.Git
			if g == nil || (g.Branch == "" && !g.Detached) {
				return nil
			}
			name := g.Branch
			if g.Detached {
				name = "detached"
			}
			return badge(true, render.Truncate(name, e.Opts.Int("max_len", 32)), "branch")
		}})
	register(Def{ID: "git_status", Desc: "Staged / modified / ahead / behind", Claude: true, Cursor: true, Needs: NeedGit, Render: renderGitStatus})
	register(Def{ID: "worktree", Desc: "Git worktree name", Claude: true, Cursor: true,
		Render: func(e *Env) []Cell {
			name := e.Snap.GitWorktree
			if w := e.Snap.Worktree; w != nil && w.Name != "" {
				name = w.Name
			}
			if name == "" {
				return nil
			}
			return []Cell{{Text: "wt:", Role: "label"}, {Text: name, Role: "plain"}}
		}})
	register(Def{ID: "pr", Desc: "Open pull request with review state", Claude: true, Cursor: true, Needs: NeedPR, Render: renderPR})
}

func renderDir(e *Env) []Cell {
	p := e.Snap.Dirs.Current
	if p == "" {
		return nil
	}
	if home := e.Getenv("HOME"); home != "" && (p == home || strings.HasPrefix(p, home+"/")) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	if depth := e.Opts.Int("depth", 1); depth > 0 {
		parts := strings.Split(strings.Trim(p, "/"), "/")
		if len(parts) > depth {
			p = strings.Join(parts[len(parts)-depth:], "/")
		}
	}
	return []Cell{{Text: p, Role: "dir"}}
}

func renderGitStatus(e *Env) []Cell {
	g := e.Snap.Git
	if g == nil {
		return nil
	}
	var cells []Cell
	add := func(on bool, n int, prefix, role string) {
		if !on || n == 0 {
			return
		}
		sp := ""
		if len(cells) > 0 {
			sp = " "
		}
		cells = append(cells, Cell{Text: fmt.Sprintf("%s%s%d", sp, prefix, n), Role: role})
	}
	ab := e.Opts.Bool("ahead_behind", true)
	add(e.Opts.Bool("staged", true), g.Staged, "+", "ok")
	add(e.Opts.Bool("modified", true), g.Modified, "~", "warn")
	add(ab, g.Ahead, "↑", "accent")
	add(ab, g.Behind, "↓", "accent")
	return cells
}

func renderPR(e *Env) []Cell {
	pr := e.Snap.PR
	if pr == nil || pr.Number == 0 {
		return nil
	}
	num := fmt.Sprintf("#%d", pr.Number)
	if pr.Kind == "mr" {
		num = fmt.Sprintf("!%d", pr.Number)
	}
	var cells []Cell
	if pr.URL != "" && e.Level != render.LevelNone && linksOn(e) {
		cells = append(cells, Cell{Raw: true,
			Text: "\x1b]8;;" + pr.URL + "\x07" + e.Theme.Style("accent").Paint(num, e.Level) + "\x1b]8;;\x07"})
	} else {
		cells = append(cells, Cell{Text: num, Role: "accent"})
	}
	switch pr.ReviewState {
	case "approved":
		cells = append(cells, Cell{Text: " ✓", Role: "ok"})
	case "changes_requested":
		cells = append(cells, Cell{Text: " ✗", Role: "crit"})
	case "draft":
		cells = append(cells, Cell{Text: " draft", Role: "dim"})
	}
	return cells
}

// linksOn decides OSC 8 hyperlinks: option link=always|never|auto (terminal allow-list).
func linksOn(e *Env) bool {
	switch e.Opts.String("link", "auto") {
	case "always":
		return true
	case "never":
		return false
	}
	if e.Getenv("FORCE_HYPERLINK") == "1" {
		return true
	}
	switch e.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "ghostty", "vscode":
		return true
	}
	return strings.Contains(e.Getenv("TERM"), "kitty")
}
