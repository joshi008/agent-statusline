package engine

import (
	"context"
	"errors"
	"strings"
	"time"
)

// PreviewNow is the clock used by preview and golden tests (fixture reset times are after it).
var PreviewNow = time.Unix(1790000000, 0)

const previewGitStatus = `# branch.oid 1234abcd
# branch.head main
# branch.upstream origin/main
# branch.ab +1 -0
1 M. N... 100644 100644 100644 aaa bbb internal/engine/engine.go
1 A. N... 000000 100644 100644 000 bbb internal/engine/preview.go
1 .M N... 100644 100644 100644 aaa bbb README.md
? scratch.txt
`

// PreviewRunner answers git/gh/sh with canned output so previews are deterministic and offline.
func PreviewRunner(_ context.Context, _ string, name string, args ...string) (string, error) {
	joined := name + " " + strings.Join(args, " ")
	switch {
	case name == "git" && strings.Contains(joined, " status "):
		return previewGitStatus, nil
	case name == "git" && strings.Contains(joined, "remote get-url"):
		return "git@github.com:joshi008/status-bar.git\n", nil
	case name == "gh":
		return `{"number":7,"url":"https://github.com/joshi008/status-bar/pull/7","reviewDecision":"APPROVED","isDraft":false}`, nil
	case name == "sh":
		return "custom\n", nil
	}
	return "", errors.New("preview: unexpected command " + joined)
}
