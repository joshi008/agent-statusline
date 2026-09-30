package harness

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	require.Equal(t, model.HarnessCursor, Detect(fixtures.Read("cursor-full")))
	require.Equal(t, model.HarnessCursor, Detect(fixtures.Read("cursor-minimal")))
	require.Equal(t, model.HarnessClaude, Detect(fixtures.Read("claude-full")))
	require.Equal(t, model.HarnessClaude, Detect(fixtures.Read("claude-minimal")))
	require.Equal(t, model.HarnessClaude, Detect([]byte(`{"model":{"display_name":"x"}}`)))
}

func TestParse_Explicit(t *testing.T) {
	s, err := Parse(fixtures.Read("cursor-full"), "cursor", func(string) string { return "" })
	require.NoError(t, err)
	require.Equal(t, model.HarnessCursor, s.Harness)
	s, err = Parse(fixtures.Read("claude-full"), "auto", func(string) string { return "" })
	require.NoError(t, err)
	require.Equal(t, model.HarnessClaude, s.Harness)
	_, err = Parse(fixtures.Read("claude-full"), "vim", func(string) string { return "" })
	require.Error(t, err)
}
