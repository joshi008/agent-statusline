package harness

import (
	"encoding/json"
	"fmt"

	"github.com/joshi008/agent-statusline/internal/model"
)

// Detect guesses the harness from payload keys. Cursor-only keys win; otherwise Claude.
func Detect(data []byte) model.Harness {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return model.HarnessClaude
	}
	if _, ok := keys["render_width_chars"]; ok {
		return model.HarnessCursor
	}
	if _, ok := keys["autorun"]; ok {
		return model.HarnessCursor
	}
	return model.HarnessClaude
}

// Parse dispatches on want ("claude", "cursor", "auto").
func Parse(data []byte, want string, env func(string) string) (*model.Snapshot, error) {
	h := model.Harness(want)
	if want == "auto" || want == "" {
		h = Detect(data)
	}
	switch h {
	case model.HarnessClaude:
		return ParseClaude(data, env)
	case model.HarnessCursor:
		return ParseCursor(data)
	default:
		return nil, fmt.Errorf("unknown harness %q (want claude, cursor or auto)", want)
	}
}
