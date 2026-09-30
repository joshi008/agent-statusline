package enrich

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/cache"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/segments"
	"github.com/stretchr/testify/require"
)

type resp struct {
	out   string
	err   error
	delay time.Duration
}

// fake answers by "<name> <first non-flag arg>", e.g. "git status", "git remote", "gh pr", "sh <cmd>".
type fake struct {
	mu    sync.Mutex
	resp  map[string]resp
	calls []string
}

func (f *fake) run(ctx context.Context, dir, name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			key += " " + a
			break
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, key)
	r, ok := f.resp[key]
	f.mu.Unlock()
	if !ok {
		return "", errors.New("unexpected " + key)
	}
	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return r.out, r.err
}

func (f *fake) count(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == key {
			n++
		}
	}
	return n
}

func snap(h model.Harness) *model.Snapshot {
	return &model.Snapshot{Harness: h, SessionID: "s", Dirs: model.Dirs{Current: "/repo"}}
}

func TestGit_Parses(t *testing.T) {
	f := &fake{resp: map[string]resp{"git status": {out: porcelain}}}
	s := snap(model.HarnessClaude)
	Enrich(context.Background(), s, Request{Needs: segments.NeedGit, Run: f.run})
	require.Equal(t, "main", s.Git.Branch)
}

func TestGit_NotARepo(t *testing.T) {
	f := &fake{resp: map[string]resp{"git status": {err: &ExitError{Code: 128, Stderr: "not a git repository"}}}}
	c := cache.New(t.TempDir(), "s")
	for i := 0; i < 2; i++ {
		s := snap(model.HarnessClaude)
		Enrich(context.Background(), s, Request{Needs: segments.NeedGit, Run: f.run, Cache: c})
		require.Nil(t, s.Git)
	}
	require.Equal(t, 1, f.count("git status"), "not-a-repo is a cached answer")
}

func TestGit_TimeoutKeepsRenderFast(t *testing.T) {
	f := &fake{resp: map[string]resp{"git status": {out: porcelain, delay: 3 * time.Second}}}
	s := snap(model.HarnessClaude)
	start := time.Now()
	Enrich(context.Background(), s, Request{Needs: segments.NeedGit, Run: f.run})
	require.Less(t, time.Since(start), Timeout+300*time.Millisecond)
	require.Nil(t, s.Git)
}

func TestRepo_OnlyWhenPayloadLacksIt(t *testing.T) {
	f := &fake{resp: map[string]resp{"git remote": {out: "git@github.com:a/b.git\n"}}}
	s := snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedRepo, Run: f.run})
	require.Equal(t, "b", s.Repo.Name)

	s = snap(model.HarnessClaude)
	s.Repo = &model.Repo{Name: "payload"}
	Enrich(context.Background(), s, Request{Needs: segments.NeedRepo, Run: f.run})
	require.Equal(t, "payload", s.Repo.Name)
	require.Equal(t, 1, f.count("git remote"))
}

func TestPR_OnlyCursorAndOnlyWhenMissing(t *testing.T) {
	f := &fake{resp: map[string]resp{"gh pr": {out: `{"number":7,"url":"u","reviewDecision":"APPROVED"}`}}}
	s := snap(model.HarnessClaude)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: f.run})
	require.Nil(t, s.PR, "Claude's payload is authoritative: no PR means no PR")
	require.Zero(t, f.count("gh pr"))

	s = snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: f.run})
	require.Equal(t, 7, s.PR.Number)
}

func TestPR_NoPR(t *testing.T) {
	f := &fake{resp: map[string]resp{"gh pr": {err: &ExitError{Code: 1, Stderr: "no pull requests found"}}}}
	s := snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: f.run})
	require.Nil(t, s.PR)
}

func TestPR_GhFails(t *testing.T) {
	now := time.Unix(1000, 0)
	c := cache.NewWithClock(t.TempDir(), "s", func() time.Time { return now })
	ok := &fake{resp: map[string]resp{"gh pr": {out: `{"number":9}`}}}
	s := snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: ok.run, Cache: c})
	require.Equal(t, 9, s.PR.Number)

	now = now.Add(time.Hour)
	broken := &fake{resp: map[string]resp{"gh pr": {err: errors.New("exec: gh: not found")}}}
	s = snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: broken.run, Cache: c})
	require.Equal(t, 9, s.PR.Number, "stale PR beats a blank segment")

	s = snap(model.HarnessCursor)
	Enrich(context.Background(), s, Request{Needs: segments.NeedPR, Run: broken.run})
	require.Nil(t, s.PR, "no cache and gh failing: segment is simply skipped")
}

func TestCustom_TextAndCmd(t *testing.T) {
	f := &fake{resp: map[string]resp{"sh kubectl config current-context": {out: "prod-eu\nextra\n"}}}
	s := snap(model.HarnessClaude)
	Enrich(context.Background(), s, Request{Run: f.run, Custom: []config.Custom{
		{ID: "env", Text: "prod"},
		{ID: "k8s", Run: "kubectl config current-context"},
	}})
	require.Equal(t, "prod", s.Custom["env"])
	require.Equal(t, "prod-eu", s.Custom["k8s"], "first line only")
}

func TestNoDirSkipsCommands(t *testing.T) {
	f := &fake{resp: map[string]resp{}}
	s := snap(model.HarnessCursor)
	s.Dirs.Current = ""
	Enrich(context.Background(), s, Request{Needs: segments.NeedGit | segments.NeedRepo | segments.NeedPR, Run: f.run})
	require.Empty(t, f.calls)
}
