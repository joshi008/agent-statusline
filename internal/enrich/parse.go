package enrich

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/joshi008/agent-statusline/internal/model"
)

// ParseGitStatus reads `git status --porcelain=v2 --branch`. Empty input (not a repo) is nil.
func ParseGitStatus(out string) *model.Git {
	if strings.TrimSpace(out) == "" {
		return nil
	}
	g := &model.Git{}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			if h := strings.TrimPrefix(line, "# branch.head "); h == "(detached)" {
				g.Detached = true
			} else {
				g.Branch = h
			}
		case strings.HasPrefix(line, "# branch.ab "):
			fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &g.Ahead, &g.Behind)
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			if len(line) >= 4 {
				if line[2] != '.' {
					g.Staged++
				}
				if line[3] != '.' {
					g.Modified++
				}
			}
		case strings.HasPrefix(line, "u "):
			g.Modified++
		}
	}
	return g
}

// ParseRemote understands scp-style (git@host:owner/name), https:// and ssh:// remotes.
func ParseRemote(raw string) *model.Repo {
	u := strings.TrimSuffix(strings.TrimSpace(raw), ".git")
	var host, p string
	switch {
	case strings.Contains(u, "://"):
		pu, err := url.Parse(u)
		if err != nil {
			return nil
		}
		host, p = pu.Hostname(), pu.Path
	case strings.Contains(u, ":") && !strings.HasPrefix(u, "/"):
		i := strings.Index(u, ":")
		host, p = u[:i], u[i+1:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
	default:
		return nil
	}
	p = strings.Trim(p, "/")
	i := strings.LastIndex(p, "/")
	if host == "" || i <= 0 {
		return nil
	}
	return &model.Repo{Host: host, Owner: p[:i], Name: p[i+1:]}
}

// ParsePR reads `gh pr view --json number,url,reviewDecision,isDraft`.
func ParsePR(out string) *model.PR {
	var r struct {
		Number         int    `json:"number"`
		URL            string `json:"url"`
		ReviewDecision string `json:"reviewDecision"`
		IsDraft        bool   `json:"isDraft"`
	}
	if json.Unmarshal([]byte(out), &r) != nil || r.Number == 0 {
		return nil
	}
	state := "pending"
	switch r.ReviewDecision {
	case "APPROVED":
		state = "approved"
	case "CHANGES_REQUESTED":
		state = "changes_requested"
	}
	if r.IsDraft {
		state = "draft"
	}
	return &model.PR{Number: r.Number, URL: r.URL, ReviewState: state}
}
