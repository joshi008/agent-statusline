// Package harness parses each tool's stdin payload into a model.Snapshot.
package harness

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/joshi008/agent-statusline/internal/model"
)

type rawModel struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	ParamSummary string `json:"param_summary"`
	MaxMode      bool   `json:"max_mode"`
}

type rawRepo struct{ Host, Owner, Name string }

type rawWorkspace struct {
	CurrentDir  string   `json:"current_dir"`
	ProjectDir  string   `json:"project_dir"`
	AddedDirs   []string `json:"added_dirs"`
	GitWorktree string   `json:"git_worktree"`
	Repo        *rawRepo `json:"repo"`
}

type rawUsage struct {
	Input       int64 `json:"input_tokens"`
	Output      int64 `json:"output_tokens"`
	CacheCreate int64 `json:"cache_creation_input_tokens"`
	CacheRead   int64 `json:"cache_read_input_tokens"`
}

type rawContext struct {
	TotalInput   int64     `json:"total_input_tokens"`
	TotalOutput  int64     `json:"total_output_tokens"`
	Size         *int64    `json:"context_window_size"`
	UsedPct      *float64  `json:"used_percentage"`
	RemainingPct *float64  `json:"remaining_percentage"`
	Usage        *rawUsage `json:"current_usage"`
}

type rawWorktree struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	OriginalCwd    string `json:"original_cwd"`
	OriginalBranch string `json:"original_branch"`
}

type rawLimit struct {
	UsedPct  float64 `json:"used_percentage"`
	ResetsAt *int64  `json:"resets_at"`
}

type rawClaude struct {
	SessionID   string       `json:"session_id"`
	SessionName string       `json:"session_name"`
	Version     string       `json:"version"`
	Cwd         string       `json:"cwd"`
	Model       rawModel     `json:"model"`
	Workspace   rawWorkspace `json:"workspace"`
	OutputStyle *struct {
		Name string `json:"name"`
	} `json:"output_style"`
	Cost *struct {
		USD           float64 `json:"total_cost_usd"`
		DurationMs    int64   `json:"total_duration_ms"`
		APIDurationMs int64   `json:"total_api_duration_ms"`
		LinesAdded    int     `json:"total_lines_added"`
		LinesRemoved  int     `json:"total_lines_removed"`
	} `json:"cost"`
	Context     *rawContext `json:"context_window"`
	PromptCache *struct {
		Warm      bool     `json:"warm"`
		HitRatio  *float64 `json:"hit_ratio"`
		TTL       string   `json:"ttl"`
		ExpiresAt *int64   `json:"expires_at"`
		Requests  int      `json:"requests"`
		Misses    int      `json:"misses"`
	} `json:"prompt_cache"`
	FastMode *bool `json:"fast_mode"`
	Effort   *struct {
		Level string `json:"level"`
	} `json:"effort"`
	Thinking *struct {
		Enabled bool `json:"enabled"`
	} `json:"thinking"`
	RateLimits map[string]rawLimit `json:"rate_limits"`
	Vim        *struct {
		Mode string `json:"mode"`
	} `json:"vim"`
	Agent *struct {
		Name string `json:"name"`
	} `json:"agent"`
	PR *struct {
		Number      int    `json:"number"`
		URL         string `json:"url"`
		ReviewState string `json:"review_state"`
		Kind        string `json:"kind"`
	} `json:"pr"`
	Worktree *rawWorktree `json:"worktree"`
}

var limitKeys = map[string]string{"five_hour": "5h", "seven_day": "7d", "spend_limit": "spend"}

// ParseClaude converts a Claude Code status line payload. env supplies COLUMNS.
func ParseClaude(data []byte, env func(string) string) (*model.Snapshot, error) {
	var r rawClaude
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	s := &model.Snapshot{
		Harness:     model.HarnessClaude,
		SessionID:   r.SessionID,
		SessionName: r.SessionName,
		Version:     r.Version,
		Model:       model.ModelInfo{ID: r.Model.ID, DisplayName: r.Model.DisplayName},
		FastMode:    r.FastMode,
		Limits:      map[string]model.Limit{},
		GitWorktree: r.Workspace.GitWorktree,
		Now:         time.Now(),
	}
	if w, err := strconv.Atoi(env("COLUMNS")); err == nil && w > 0 {
		s.Width = w
	}
	s.Dirs = dirsFrom(r.Cwd, r.Workspace)
	if r.Workspace.Repo != nil {
		s.Repo = &model.Repo{Host: r.Workspace.Repo.Host, Owner: r.Workspace.Repo.Owner, Name: r.Workspace.Repo.Name}
	}
	if r.OutputStyle != nil {
		s.OutputStyle = r.OutputStyle.Name
	}
	if r.Cost != nil {
		s.Cost = &model.Cost{USD: r.Cost.USD, DurationMs: r.Cost.DurationMs, APIDurationMs: r.Cost.APIDurationMs,
			LinesAdded: r.Cost.LinesAdded, LinesRemoved: r.Cost.LinesRemoved}
	}
	s.Context = contextFrom(r.Context)
	if r.PromptCache != nil {
		s.Cache = &model.PromptCache{Warm: r.PromptCache.Warm, HitRatio: r.PromptCache.HitRatio, TTL: r.PromptCache.TTL,
			ExpiresAt: r.PromptCache.ExpiresAt, Requests: r.PromptCache.Requests, Misses: r.PromptCache.Misses}
	}
	if r.Effort != nil {
		s.Effort = r.Effort.Level
	}
	if r.Thinking != nil {
		v := r.Thinking.Enabled
		s.Thinking = &v
	}
	for k, v := range r.RateLimits {
		if short, ok := limitKeys[k]; ok {
			s.Limits[short] = model.Limit{UsedPct: v.UsedPct, ResetsAt: v.ResetsAt}
		}
	}
	if r.Vim != nil {
		s.Vim = r.Vim.Mode
	}
	if r.Agent != nil {
		s.Agent = r.Agent.Name
	}
	if r.PR != nil {
		s.PR = &model.PR{Number: r.PR.Number, URL: r.PR.URL, ReviewState: r.PR.ReviewState, Kind: r.PR.Kind}
	}
	s.Worktree = worktreeFrom(r.Worktree)
	return s, nil
}

func dirsFrom(cwd string, w rawWorkspace) model.Dirs {
	d := model.Dirs{Current: w.CurrentDir, Project: w.ProjectDir, Added: w.AddedDirs}
	if d.Current == "" {
		d.Current = cwd
	}
	if d.Project == "" {
		d.Project = d.Current
	}
	return d
}

func contextFrom(r *rawContext) *model.ContextWindow {
	if r == nil {
		return nil
	}
	c := &model.ContextWindow{UsedPct: r.UsedPct, RemainingPct: r.RemainingPct, Size: r.Size,
		TotalInput: r.TotalInput, TotalOutput: r.TotalOutput}
	if r.Usage != nil {
		c.Usage = &model.Usage{Input: r.Usage.Input, Output: r.Usage.Output, CacheCreate: r.Usage.CacheCreate, CacheRead: r.Usage.CacheRead}
	}
	return c
}

func worktreeFrom(r *rawWorktree) *model.Worktree {
	if r == nil {
		return nil
	}
	return &model.Worktree{Name: r.Name, Path: r.Path, Branch: r.Branch, OriginalCwd: r.OriginalCwd, OriginalBranch: r.OriginalBranch}
}
