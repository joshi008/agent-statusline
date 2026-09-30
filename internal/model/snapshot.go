// Package model holds the harness-agnostic view of a session that segments render.
// Absent strings are "", absent numbers are nil pointers, absent objects are nil.
package model

import "time"

type Harness string

const (
	HarnessClaude Harness = "claude"
	HarnessCursor Harness = "cursor"
)

type ModelInfo struct {
	ID           string
	DisplayName  string
	ParamSummary string // Cursor only: e.g. "effort high"
	MaxMode      bool   // Cursor only
}

type Usage struct{ Input, Output, CacheCreate, CacheRead int64 }

type ContextWindow struct {
	UsedPct      *float64
	RemainingPct *float64
	Size         *int64
	TotalInput   int64
	TotalOutput  int64
	Usage        *Usage
}

type PromptCache struct {
	Warm      bool
	HitRatio  *float64
	TTL       string
	ExpiresAt *int64
	Requests  int
	Misses    int
}

type Cost struct {
	USD           float64
	DurationMs    int64
	APIDurationMs int64
	LinesAdded    int
	LinesRemoved  int
}

// Limit is one rate-limit window. Keys in Snapshot.Limits: "5h", "7d", "spend".
type Limit struct {
	UsedPct  float64
	ResetsAt *int64
}

type Dirs struct {
	Current string
	Project string
	Added   []string
}

type Repo struct{ Host, Owner, Name string }

type Worktree struct{ Name, Path, Branch, OriginalCwd, OriginalBranch string }

// PR review states: approved | pending | changes_requested | draft. Kind "mr" for GitLab.
type PR struct {
	Number      int
	URL         string
	ReviewState string
	Kind        string
}

// Git is filled by enrichment (Task 10), never by the payload.
type Git struct {
	Branch   string
	Detached bool
	Staged   int
	Modified int
	Ahead    int
	Behind   int
}

type Snapshot struct {
	Harness     Harness
	Width       int // 0 = unknown
	SessionID   string
	SessionName string
	Version     string
	Model       ModelInfo
	Effort      string // low|medium|high|xhigh|max, "" when unknown
	Thinking    *bool
	FastMode    *bool
	Autorun     *bool
	Context     *ContextWindow
	Cache       *PromptCache
	Cost        *Cost
	Limits      map[string]Limit
	Dirs        Dirs
	Repo        *Repo
	GitWorktree string
	Worktree    *Worktree
	PR          *PR
	Agent       string
	Vim         string
	OutputStyle string
	Git         *Git
	Custom      map[string]string // custom segment output by id, filled by enrichment (Task 10)
	Now         time.Time
}
