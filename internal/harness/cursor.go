package harness

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/joshi008/agent-statusline/internal/model"
)

type rawCursor struct {
	SessionID   string       `json:"session_id"`
	SessionName string       `json:"session_name"`
	Version     string       `json:"version"`
	Cwd         string       `json:"cwd"`
	RenderWidth int          `json:"render_width_chars"`
	Autorun     *bool        `json:"autorun"`
	Model       rawModel     `json:"model"`
	Workspace   rawWorkspace `json:"workspace"`
	OutputStyle *struct {
		Name string `json:"name"`
	} `json:"output_style"`
	Context *rawContext `json:"context_window"`
	Vim     *struct {
		Mode string `json:"mode"`
	} `json:"vim"`
	Worktree *rawWorktree `json:"worktree"`
}

var effortRe = regexp.MustCompile(`\beffort (low|medium|high|xhigh|max)\b`)

// ParseCursor converts a Cursor CLI status line payload. Width comes from render_width_chars.
func ParseCursor(data []byte) (*model.Snapshot, error) {
	var r rawCursor
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	s := &model.Snapshot{
		Harness:     model.HarnessCursor,
		Width:       r.RenderWidth,
		SessionID:   r.SessionID,
		SessionName: r.SessionName,
		Version:     r.Version,
		Model:       model.ModelInfo{ID: r.Model.ID, DisplayName: r.Model.DisplayName, ParamSummary: r.Model.ParamSummary, MaxMode: r.Model.MaxMode},
		Autorun:     r.Autorun,
		Limits:      map[string]model.Limit{},
		Dirs:        dirsFrom(r.Cwd, r.Workspace),
		Context:     contextFrom(r.Context),
		Worktree:    worktreeFrom(r.Worktree),
		Now:         time.Now(),
	}
	if m := effortRe.FindStringSubmatch(r.Model.ParamSummary); m != nil {
		s.Effort = m[1]
	}
	if r.OutputStyle != nil {
		s.OutputStyle = r.OutputStyle.Name
	}
	if r.Vim != nil {
		s.Vim = r.Vim.Mode
	}
	return s, nil
}
