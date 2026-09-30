package engine

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/stretchr/testify/require"
)

const subagentInput = `{"session_id":"s","columns":0,"tasks":[
 {"id":"t1","name":"explorer","type":"Explore","status":"running","description":"find files","label":"Explore",
  "startTime":1789999880000,"model":"claude-opus-5-5","effort":"high","contextWindowSize":200000,"tokenCount":74000},
 {"id":"t2","name":"","label":"planner","status":"running","startTime":0,"tokenCount":0,"effort":8000},
 {"id":"","name":"skip-me"}
]}`

func TestRenderSubagents_Default(t *testing.T) {
	out := RenderSubagents([]byte(subagentInput), config.Default(), render.LevelNone, PreviewNow)
	require.Equal(t, `{"id":"t1","content":"explorer opus-5-5 37% 74k"}`+"\n"+`{"id":"t2","content":"planner"}`+"\n", out)
}

func TestRenderSubagents_CustomFormatAndWidth(t *testing.T) {
	cfg := config.Default()
	cfg.Claude.Subagents.Format = "{name} [{effort}] {elapsed} {status} {unknown}"
	out := RenderSubagents([]byte(subagentInput), cfg, render.LevelNone, PreviewNow)
	require.Contains(t, out, `"content":"explorer [high] 2m0s running {unknown}"`)
	require.Contains(t, out, `"content":"planner [8000] running {unknown}"`)

	narrow := []byte(`{"columns":10,"tasks":[{"id":"t1","name":"explorer","model":"claude-opus-5-5","contextWindowSize":200000,"tokenCount":74000}]}`)
	out = RenderSubagents(narrow, config.Default(), render.LevelNone, PreviewNow)
	require.Equal(t, `{"id":"t1","content":"explorer …"}`+"\n", out)
}

func TestRenderSubagents_ColourAndOff(t *testing.T) {
	out := RenderSubagents([]byte(subagentInput), config.Default(), render.LevelBasic, PreviewNow)
	require.Contains(t, out, `\u001b[1mexplorer`)
	cfg := config.Default()
	cfg.Claude.Subagents.Enabled = false
	require.Equal(t, "", RenderSubagents([]byte(subagentInput), cfg, render.LevelNone, PreviewNow))
	require.Equal(t, "", RenderSubagents([]byte("nope"), config.Default(), render.LevelNone, PreviewNow))
}

// Final review, MINOR C: one task with a mistyped field is skipped; the others still render.
func TestRenderSubagents_BadTaskDoesNotBlankTheRest(t *testing.T) {
	in := `{"columns":0,"tasks":[
 {"id":"bad","name":"broken","contextWindowSize":"200k"},
 {"id":"t1","name":"explorer","model":"claude-opus-5-5","contextWindowSize":200000,"tokenCount":74000}
]}`
	out := RenderSubagents([]byte(in), config.Default(), render.LevelNone, PreviewNow)
	require.Equal(t, `{"id":"t1","content":"explorer opus-5-5 37% 74k"}`+"\n", out)
}
