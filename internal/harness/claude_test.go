package harness

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParseClaude_Full(t *testing.T) {
	s, err := ParseClaude(fixtures.Read("claude-full"), envOf(map[string]string{"COLUMNS": "120"}))
	require.NoError(t, err)
	require.Equal(t, model.HarnessClaude, s.Harness)
	require.Equal(t, 120, s.Width)
	require.Equal(t, "status-bar research", s.SessionName)
	require.Equal(t, "claude-fable-5-1[1m]", s.Model.ID)
	require.Equal(t, "Fable 5.1", s.Model.DisplayName)
	require.Equal(t, "high", s.Effort)
	require.True(t, *s.Thinking)
	require.False(t, *s.FastMode)
	require.InDelta(t, 37.2, *s.Context.UsedPct, 0.001)
	require.EqualValues(t, 1000000, *s.Context.Size)
	require.EqualValues(t, 358000, s.Context.Usage.CacheRead)
	require.InDelta(t, 1.2345, s.Cost.USD, 0.0001)
	require.Equal(t, 156, s.Cost.LinesAdded)
	require.InDelta(t, 42, s.Limits["5h"].UsedPct, 0.001)
	require.EqualValues(t, 1790001400, *s.Limits["5h"].ResetsAt)
	require.InDelta(t, 71, s.Limits["7d"].UsedPct, 0.001)
	_, hasSpend := s.Limits["spend"]
	require.False(t, hasSpend)
	require.True(t, s.Cache.Warm)
	require.InDelta(t, 0.91, *s.Cache.HitRatio, 0.001)
	require.Equal(t, "status-bar", s.Repo.Name)
	require.Equal(t, 42, s.PR.Number)
	require.Equal(t, "pending", s.PR.ReviewState)
	require.Equal(t, "/Users/example/Projects/status-bar", s.Dirs.Current)
	require.Equal(t, []string{"/tmp"}, s.Dirs.Added)
	require.Equal(t, "2.1.283", s.Version)
	require.Equal(t, "default", s.OutputStyle)
	require.False(t, s.Now.IsZero())
}

func TestParseClaude_Minimal_NullsAndAbsent(t *testing.T) {
	s, err := ParseClaude(fixtures.Read("claude-minimal"), envOf(nil))
	require.NoError(t, err)
	require.Equal(t, 0, s.Width)
	require.Empty(t, s.SessionName)
	require.Nil(t, s.Context.UsedPct)
	require.Nil(t, s.Context.Usage)
	require.Empty(t, s.Limits)
	require.Nil(t, s.Cache)
	require.Nil(t, s.PR)
	require.Nil(t, s.Repo)
	require.Nil(t, s.Worktree)
	require.Empty(t, s.Agent)
	require.Empty(t, s.Vim)
	require.NotNil(t, s.Cost)
	require.Equal(t, 0.0, s.Cost.USD)
}

func TestParseClaude_WorktreeAndVimAndAgent(t *testing.T) {
	data := []byte(`{"model":{"display_name":"Opus"},"vim":{"mode":"NORMAL"},"agent":{"name":"rev"},
	  "workspace":{"current_dir":"/x","git_worktree":"feat"},
	  "worktree":{"name":"wt","path":"/p","branch":"b","original_cwd":"/o","original_branch":"main"},
	  "rate_limits":{"spend_limit":{"used_percentage":120.5,"resets_at":5}}}`)
	s, err := ParseClaude(data, envOf(nil))
	require.NoError(t, err)
	require.Equal(t, "NORMAL", s.Vim)
	require.Equal(t, "rev", s.Agent)
	require.Equal(t, "feat", s.GitWorktree)
	require.Equal(t, "main", s.Worktree.OriginalBranch)
	require.InDelta(t, 120.5, s.Limits["spend"].UsedPct, 0.001)
}

func TestParseClaude_InvalidJSON(t *testing.T) {
	_, err := ParseClaude([]byte("{nope"), envOf(nil))
	require.Error(t, err)
}
