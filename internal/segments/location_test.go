package segments

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	s := &model.Snapshot{Dirs: model.Dirs{Current: "/Users/me/Desktop/Projects/status-bar"}}
	require.Equal(t, "status-bar", run(t, "dir", newEnv(t, s, nil)))
	require.Equal(t, "Projects/status-bar", run(t, "dir", newEnv(t, s, Options{"depth": 2})))
	require.Equal(t, "~/Desktop/Projects/status-bar", run(t, "dir", newEnv(t, s, Options{"depth": 0})))
	s.Dirs.Current = "/Users/me"
	require.Equal(t, "~", run(t, "dir", newEnv(t, s, nil)))
	s.Dirs.Current = "/"
	require.Equal(t, "/", run(t, "dir", newEnv(t, s, nil)))
	s.Dirs.Current = ""
	require.Equal(t, "", run(t, "dir", newEnv(t, s, nil)))
}

func TestRepo(t *testing.T) {
	s := &model.Snapshot{Repo: &model.Repo{Host: "github.com", Owner: "joshi008", Name: "status-bar"}}
	require.Equal(t, "joshi008/status-bar", run(t, "repo", newEnv(t, s, nil)))
	require.Equal(t, "status-bar", run(t, "repo", newEnv(t, s, Options{"name_only": true})))
}

func TestBranchAndGitStatus(t *testing.T) {
	s := &model.Snapshot{Git: &model.Git{Branch: "feature/a-very-long-branch-name", Staged: 2, Modified: 5, Ahead: 1}}
	require.Equal(t, "feature/a-very-long-branch-name", run(t, "branch", newEnv(t, s, nil)))
	require.Equal(t, "feature/a…", run(t, "branch", newEnv(t, s, Options{"max_len": 10})))
	require.Equal(t, "+2 ~5 ↑1", run(t, "git_status", newEnv(t, s, nil)))
	require.Equal(t, "~5 ↑1", run(t, "git_status", newEnv(t, s, Options{"staged": false})))
	s.Git = &model.Git{Detached: true}
	require.Equal(t, "detached", run(t, "branch", newEnv(t, s, nil)))
	require.Equal(t, "", run(t, "git_status", newEnv(t, s, nil)), "clean tree renders nothing")
	s.Git = nil
	require.Equal(t, "", run(t, "branch", newEnv(t, s, nil)))
}

func TestWorktree(t *testing.T) {
	s := &model.Snapshot{GitWorktree: "linked"}
	require.Equal(t, "wt:linked", run(t, "worktree", newEnv(t, s, nil)))
	s.Worktree = &model.Worktree{Name: "session-wt"}
	require.Equal(t, "wt:session-wt", run(t, "worktree", newEnv(t, s, nil)))
}

func TestPR(t *testing.T) {
	s := &model.Snapshot{PR: &model.PR{Number: 42, URL: "https://x/pull/42", ReviewState: "approved"}}
	require.Equal(t, "#42 ✓", run(t, "pr", newEnv(t, s, nil)))
	s.PR.ReviewState = "changes_requested"
	require.Equal(t, "#42 ✗", run(t, "pr", newEnv(t, s, nil)))
	s.PR.ReviewState, s.PR.Kind = "draft", "mr"
	require.Equal(t, "!42 draft", run(t, "pr", newEnv(t, s, nil)))
	s.PR.ReviewState = "pending"
	require.Equal(t, "!42", run(t, "pr", newEnv(t, s, nil)))
}

func TestPR_OSC8Links(t *testing.T) {
	s := &model.Snapshot{PR: &model.PR{Number: 7, URL: "https://x/pull/7"}}
	e := newEnv(t, s, nil)
	e.Level = render.LevelTrue
	d, _ := Lookup("pr")
	// Apple Terminal: no TERM_PROGRAM match → plain text, no OSC 8.
	require.NotContains(t, d.Render(e)[0].Text, "\x1b]8;;")
	e.Getenv = func(k string) string { return map[string]string{"TERM_PROGRAM": "iTerm.app"}[k] }
	cell := d.Render(e)[0]
	require.True(t, cell.Raw)
	require.Contains(t, cell.Text, "\x1b]8;;https://x/pull/7\x07")
	require.Equal(t, "#7", render.Strip(cell.Text))
	e.Opts = Options{"link": "never"}
	require.NotContains(t, d.Render(e)[0].Text, "\x1b]8;;")
}
