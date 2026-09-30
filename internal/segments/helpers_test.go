package segments

import (
	"strings"
	"testing"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/theme"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func newEnv(t *testing.T, s *model.Snapshot, opts Options) *Env {
	t.Helper()
	th, err := theme.Resolve("default", nil)
	require.NoError(t, err)
	if s.Limits == nil {
		s.Limits = map[string]model.Limit{}
	}
	env := map[string]string{"HOME": "/Users/me"}
	return &Env{Snap: s, Theme: th, Level: render.LevelNone, Warn: 50, Crit: 80, Countdown: true,
		Opts: opts, Getenv: func(k string) string { return env[k] }}
}

func text(cells []Cell) string {
	var b strings.Builder
	for _, c := range cells {
		b.WriteString(render.Strip(c.Text))
	}
	return b.String()
}

func run(t *testing.T, id string, e *Env) string {
	t.Helper()
	d, ok := Lookup(id)
	require.True(t, ok, id)
	return text(d.Render(e))
}
