package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/render"
	"github.com/joshi008/agent-statusline/internal/segments"
	"github.com/joshi008/agent-statusline/internal/theme"
)

const defaultSubagentFormat = "{name} {model_short} {ctx_pct} {tokens}"

type subagentTask struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Type              string          `json:"type"`
	Status            string          `json:"status"`
	Description       string          `json:"description"`
	Label             string          `json:"label"`
	StartTime         float64         `json:"startTime"`
	Model             string          `json:"model"`
	Effort            json.RawMessage `json:"effort"`
	ContextWindowSize int64           `json:"contextWindowSize"`
	TokenCount        int64           `json:"tokenCount"`
}

type subagentRow struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

var fieldRe = regexp.MustCompile(`\{([a-z_]+)\}`)

// RenderSubagents implements Claude Code's subagentStatusLine contract.
func RenderSubagents(data []byte, cfg *config.Config, lvl render.Level, now time.Time) string {
	if cfg == nil {
		cfg = config.Default()
	}
	if !cfg.Claude.Subagents.Enabled {
		return ""
	}
	var in struct {
		Columns int               `json:"columns"`
		Tasks   []json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return ""
	}
	// Decode each task on its own so one malformed task costs only its own row.
	tasks := make([]subagentTask, 0, len(in.Tasks))
	for _, raw := range in.Tasks {
		var t subagentTask
		if json.Unmarshal(raw, &t) == nil {
			tasks = append(tasks, t)
		}
	}
	th := ResolveTheme(cfg, func(string, ...any) {})
	warn, crit := cfg.Style.Thresholds.Warn, cfg.Style.Thresholds.Crit
	format := cfg.Claude.Subagents.Format
	if format == "" {
		format = defaultSubagentFormat
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, t := range tasks {
		if t.ID == "" {
			continue
		}
		content := fieldRe.ReplaceAllStringFunc(format, func(m string) string {
			return subagentField(m[1:len(m)-1], t, th, lvl, warn, crit, now)
		})
		content = strings.Join(strings.Fields(content), " ")
		if in.Columns > 0 {
			content = render.Truncate(content, in.Columns)
		}
		_ = enc.Encode(subagentRow{ID: t.ID, Content: content})
	}
	return b.String()
}

func subagentField(name string, t subagentTask, th *theme.Theme, lvl render.Level, warn, crit float64, now time.Time) string {
	paint := func(role, s string) string { return th.Style(role).Paint(s, lvl) }
	switch name {
	case "name":
		if t.Name != "" {
			return paint("bold", t.Name)
		}
		return paint("bold", t.Label)
	case "type":
		return paint("dim", t.Type)
	case "status":
		return paint("dim", t.Status)
	case "description":
		return paint("plain", t.Description)
	case "model":
		return paint("model", t.Model)
	case "model_short":
		model := strings.TrimPrefix(t.Model, "claude-")
		model, _, _ = strings.Cut(model, "[")
		return paint("model", model)
	case "effort":
		var effort string
		if json.Unmarshal(t.Effort, &effort) != nil {
			effort = strings.TrimSpace(string(t.Effort))
		}
		if effort == "null" {
			effort = ""
		}
		return paint("effort", effort)
	case "ctx_pct":
		if t.ContextWindowSize <= 0 {
			return ""
		}
		pct := float64(t.TokenCount) / float64(t.ContextWindowSize) * 100
		return paint(segments.LevelRole(pct, warn, crit), fmt.Sprintf("%d%%", int(math.Round(pct))))
	case "tokens":
		if t.TokenCount <= 0 {
			return ""
		}
		return paint("dim", segments.Humanize(t.TokenCount))
	case "elapsed":
		ms := now.UnixMilli() - int64(t.StartTime)
		if t.StartTime <= 0 || ms < 0 {
			return ""
		}
		return paint("dim", render.Duration(ms))
	default:
		return "{" + name + "}"
	}
}
