package harness

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/fixtures"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

func TestParseCursor_Full(t *testing.T) {
	s, err := ParseCursor(fixtures.Read("cursor-full"))
	require.NoError(t, err)
	require.Equal(t, model.HarnessCursor, s.Harness)
	require.Equal(t, 156, s.Width)
	require.Equal(t, "Cursor Grok 4.6 High", s.Model.DisplayName)
	require.Equal(t, "effort high", s.Model.ParamSummary)
	require.Equal(t, "high", s.Effort)
	require.False(t, s.Model.MaxMode)
	require.False(t, *s.Autorun)
	require.InDelta(t, 30.5, *s.Context.UsedPct, 0.001)
	require.Nil(t, s.Cost)
	require.Empty(t, s.Limits)
	require.Equal(t, "status-bar research", s.SessionName)
}

func TestParseCursor_Minimal(t *testing.T) {
	s, err := ParseCursor(fixtures.Read("cursor-minimal"))
	require.NoError(t, err)
	require.True(t, s.Model.MaxMode)
	require.True(t, *s.Autorun)
	require.Empty(t, s.Effort)
	require.Nil(t, s.Context.UsedPct)
	require.Nil(t, s.Context.Size)
	require.Equal(t, "compact", s.OutputStyle)
}

func TestParseCursor_EffortFromParamSummary(t *testing.T) {
	for in, want := range map[string]string{"effort xhigh": "xhigh", "fast, effort low": "low", "fast": "", "": ""} {
		s, err := ParseCursor([]byte(`{"model":{"id":"m","display_name":"M","param_summary":"` + in + `"}}`))
		require.NoError(t, err)
		require.Equal(t, want, s.Effort, in)
	}
}
