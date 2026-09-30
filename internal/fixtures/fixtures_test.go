package fixtures

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRead_AllFixturesAreJSON(t *testing.T) {
	for _, name := range []string{"claude-full", "claude-minimal", "cursor-full", "cursor-minimal"} {
		var v map[string]any
		require.NoError(t, json.Unmarshal(Read(name), &v), name)
		require.NotEmpty(t, v["session_id"], name)
	}
}

func TestRead_UnknownPanics(t *testing.T) {
	require.Panics(t, func() { Read("nope") })
}
