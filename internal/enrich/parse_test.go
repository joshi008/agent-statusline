package enrich

import (
	"testing"

	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/stretchr/testify/require"
)

const porcelain = `# branch.oid 1234abcd
# branch.head main
# branch.upstream origin/main
# branch.ab +1 -3
1 M. N... 100644 100644 100644 aaa bbb staged.go
1 A. N... 000000 100644 100644 000 bbb added.go
1 .M N... 100644 100644 100644 aaa bbb changed.go
1 MM N... 100644 100644 100644 aaa bbb both.go
u UU N... 100644 100644 100644 100644 a b c conflict.go
? untracked.go
`

func TestParseGitStatus(t *testing.T) {
	g := ParseGitStatus(porcelain)
	require.Equal(t, &model.Git{Branch: "main", Staged: 3, Modified: 3, Ahead: 1, Behind: 3}, g)
	require.Nil(t, ParseGitStatus(""))
	d := ParseGitStatus("# branch.oid abc\n# branch.head (detached)\n")
	require.True(t, d.Detached)
	require.Empty(t, d.Branch)
}

func TestParseRemote(t *testing.T) {
	cases := map[string]*model.Repo{
		"git@github.com:joshi008/status-bar.git":            {Host: "github.com", Owner: "joshi008", Name: "status-bar"},
		"https://github.com/joshi008/status-bar":            {Host: "github.com", Owner: "joshi008", Name: "status-bar"},
		"ssh://git@gitlab.com:2222/group/sub/project.git\n": {Host: "gitlab.com", Owner: "group/sub", Name: "project"},
		"/tmp/local-repo": nil,
		"":                nil,
		"https://host/x":  nil,
	}
	for in, want := range cases {
		require.Equal(t, want, ParseRemote(in), in)
	}
}

func TestParsePR(t *testing.T) {
	require.Equal(t, &model.PR{Number: 7, URL: "u", ReviewState: "approved"},
		ParsePR(`{"number":7,"url":"u","reviewDecision":"APPROVED","isDraft":false}`))
	require.Equal(t, "changes_requested", ParsePR(`{"number":7,"reviewDecision":"CHANGES_REQUESTED"}`).ReviewState)
	require.Equal(t, "pending", ParsePR(`{"number":7,"reviewDecision":"REVIEW_REQUIRED"}`).ReviewState)
	require.Equal(t, "draft", ParsePR(`{"number":7,"isDraft":true,"reviewDecision":"APPROVED"}`).ReviewState)
	require.Nil(t, ParsePR(`{}`))
	require.Nil(t, ParsePR(``))
}
