package render

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDetectLevel(t *testing.T) {
	require.Equal(t, LevelTrue, DetectLevel(env(map[string]string{"COLORTERM": "truecolor"})))
	require.Equal(t, LevelTrue, DetectLevel(env(map[string]string{"COLORTERM": "24bit", "TERM": "xterm"})))
	require.Equal(t, Level256, DetectLevel(env(map[string]string{"TERM": "xterm-256color"})))
	require.Equal(t, LevelBasic, DetectLevel(env(map[string]string{"TERM": "xterm"})))
	require.Equal(t, LevelNone, DetectLevel(env(map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"})))
}
