// Package enrich fills Snapshot fields the payload lacks by running git / gh / custom commands,
// concurrently, each under a hard timeout and through the session cache.
package enrich

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/joshi008/agent-statusline/internal/cache"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/segments"
)

// Timeout bounds every external command.
const Timeout = 500 * time.Millisecond

type Runner func(ctx context.Context, dir, name string, args ...string) (string, error)

// ExitError means the program ran and answered with a non-zero exit (e.g. "not a git repo").
type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string { return fmt.Sprintf("exit %d: %s", e.Code, e.Stderr) }

// waitDelay bounds how long ExecRunner waits for output pipes after the process is killed.
const waitDelay = 100 * time.Millisecond

// ExecRunner runs a real process without a shell. When ctx ends, the whole process group is
// killed (see killGroupOnCancel), so a compound `sh -c` command cannot outlive the timeout.
func ExecRunner(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	killGroupOnCancel(cmd)
	cmd.WaitDelay = waitDelay
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GH_PROMPT_DISABLED=1", "NO_COLOR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) && ctx.Err() == nil {
		return string(out), &ExitError{Code: ee.ExitCode(), Stderr: strings.TrimSpace(stderr.String())}
	}
	return string(out), err
}

type Request struct {
	Needs  segments.Need
	Custom []config.Custom
	Cache  *cache.Cache
	Run    Runner
	Logf   func(format string, args ...any)
}

func (r Request) cached(key string, ttl time.Duration, fetch func() (string, error)) (string, error) {
	if r.Cache == nil {
		return fetch()
	}
	return r.Cache.Get(key, ttl, fetch)
}

// answer runs a command; an ExitError is a valid (empty) answer, anything else is a failure.
func (r Request) answer(ctx context.Context, dir, name string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	out, err := r.Run(cctx, dir, name, args...)
	var ee *ExitError
	if errors.As(err, &ee) {
		return "", nil
	}
	return out, err
}

func dirKey(prefix, dir string) string {
	h := fnv.New32a()
	h.Write([]byte(dir))
	return fmt.Sprintf("%s-%08x", prefix, h.Sum32())
}

// Enrich fills s in place. It never fails; problems are logged via r.Logf.
func Enrich(ctx context.Context, s *model.Snapshot, r Request) {
	if r.Run == nil {
		r.Run = ExecRunner
	}
	if r.Logf == nil {
		r.Logf = func(string, ...any) {}
	}
	dir := s.Dirs.Current
	var wg sync.WaitGroup
	var mu sync.Mutex
	goDo := func(f func()) {
		wg.Add(1)
		go func() { defer wg.Done(); f() }()
	}

	if r.Needs&segments.NeedGit != 0 && dir != "" {
		goDo(func() {
			out, err := r.cached(dirKey("git", dir), 5*time.Second, func() (string, error) {
				return r.answer(ctx, dir, "git", "--no-optional-locks", "status", "--porcelain=v2", "--branch")
			})
			if err != nil {
				r.Logf("git status: %v", err)
				return
			}
			g := ParseGitStatus(out)
			mu.Lock()
			s.Git = g
			mu.Unlock()
		})
	}
	if r.Needs&segments.NeedRepo != 0 && s.Repo == nil && dir != "" {
		goDo(func() {
			out, err := r.cached(dirKey("repo", dir), 5*time.Minute, func() (string, error) {
				return r.answer(ctx, dir, "git", "remote", "get-url", "origin")
			})
			if err != nil {
				r.Logf("git remote: %v", err)
				return
			}
			repo := ParseRemote(out)
			mu.Lock()
			s.Repo = repo
			mu.Unlock()
		})
	}
	if r.Needs&segments.NeedPR != 0 && s.PR == nil && s.Harness == model.HarnessCursor && dir != "" {
		goDo(func() {
			out, err := r.cached(dirKey("pr", dir), 60*time.Second, func() (string, error) {
				o, err := r.answer(ctx, dir, "gh", "pr", "view", "--json", "number,url,reviewDecision,isDraft")
				if err == nil && o == "" {
					o = "{}"
				}
				return o, err
			})
			if err != nil {
				r.Logf("gh pr view: %v", err)
				return
			}
			pr := ParsePR(out)
			mu.Lock()
			s.PR = pr
			mu.Unlock()
		})
	}
	if len(r.Custom) > 0 && s.Custom == nil {
		s.Custom = map[string]string{}
	}
	for _, c := range r.Custom {
		if c.Text != "" {
			mu.Lock() // command goroutines started earlier in this loop write the same map
			s.Custom[c.ID] = c.Text
			mu.Unlock()
			continue
		}
		c := c
		goDo(func() {
			ttl := c.TTL
			if ttl <= 0 {
				ttl = 30 * time.Second
			}
			out, err := r.cached("cmd-"+c.ID, ttl, func() (string, error) {
				return r.answer(ctx, dir, "sh", "-c", c.Run)
			})
			if err != nil {
				r.Logf("custom %s: %v", c.ID, err)
				return
			}
			first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
			mu.Lock()
			s.Custom[c.ID] = strings.TrimSpace(first)
			mu.Unlock()
		})
	}
	wg.Wait()
}
