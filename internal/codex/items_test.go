package codex

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranslate(t *testing.T) {
	ids, dropped := Translate([]string{"model", "effort", "context", "limit_5h", "limit_7d", "cost", "branch"})
	require.Equal(t, []string{"model-with-reasoning", "context-used", "five-hour-limit", "weekly-limit", "estimated-thread-cost", "git-branch"}, ids)
	require.Empty(t, dropped)

	ids, dropped = Translate([]string{"effort", "model", "context-remaining", "pr", "cmd:k8s", "model", "run-state"})
	require.Equal(t, []string{"reasoning", "model", "context-remaining", "run-state"}, ids)
	require.Equal(t, []string{"pr", "cmd:k8s"}, dropped)
}

func TestKnownItems(t *testing.T) {
	require.Len(t, KnownItems, 21)
	require.True(t, IsItem("weekly-limit"))
	require.False(t, IsItem("limit_7d"))
	require.Equal(t, `["model", "git-branch"]`, FormatArray([]string{"model", "git-branch"}))
}
