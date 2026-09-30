package codex

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// userConfig mirrors the shape of the real ~/.codex/config.toml on this machine.
const userConfig = `model = "gpt-5.6-sol"
model_reasoning_effort = "high"
approvals_reviewer = "user"
[projects."/Users/example/Projects/sample-project"]
trust_level = "trusted"

[tui]
theme = "catppuccin-latte"

[tui.model_availability_nux]
"gpt-5.6-sol" = 4

[features]
hooks = true
`

func mustSet(t *testing.T, src string, ids ...string) string {
	t.Helper()
	out, err := SetStatusLine([]byte(src), ids)
	require.NoError(t, err)
	got, ok, err := GetStatusLine(out)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, ids, got)
	return string(out)
}

func TestSet_EmptyFile(t *testing.T) {
	require.Equal(t, "[tui]\nstatus_line = [\"model\"]\n", mustSet(t, "", "model"))
}

func TestSet_UserConfigInsertsUnderTui(t *testing.T) {
	out := mustSet(t, userConfig, "model-with-reasoning", "git-branch")
	line := `status_line = ["model-with-reasoning", "git-branch"]` + "\n"
	require.Contains(t, out, "[tui]\n"+line+"theme = ")
	require.Equal(t, userConfig, strings.Replace(out, line, "", 1), "nothing else changed")
}

func TestSet_ReplaceSingleLine(t *testing.T) {
	src := "[tui]\nstatus_line = [\"model\"]\ntheme = \"x\"\n"
	require.Equal(t, "[tui]\nstatus_line = [\"git-branch\"]\ntheme = \"x\"\n", mustSet(t, src, "git-branch"))
}

func TestSet_ReplaceMultiline(t *testing.T) {
	src := "[tui]\nstatus_line = [\n  \"model\",   # first\n  \"git-branch\",\n]\ntheme = \"x\"\n"
	require.Equal(t, "[tui]\nstatus_line = [\"context-used\"]\ntheme = \"x\"\n", mustSet(t, src, "context-used"))
}

func TestCodex_PreservesComments(t *testing.T) {
	src := "# top comment\nmodel = \"gpt\" # trailing\n\n# section comment\n[tui]\n# inside tui\ntheme = \"x\" # theme comment\n"
	out := mustSet(t, src, "model")
	for _, c := range []string{"# top comment", "# trailing", "# section comment", "# inside tui", "# theme comment"} {
		require.Contains(t, out, c)
	}
	require.Equal(t, src, strings.Replace(out, "status_line = [\"model\"]\n", "", 1))
}

func TestCodex_DottedKey(t *testing.T) {
	src := "tui.status_line = [\"model\"]\n\n[projects.\"x\"]\ntrust_level = \"trusted\"\n"
	out := mustSet(t, src, "git-branch")
	require.Equal(t, "tui.status_line = [\"git-branch\"]\n\n[projects.\"x\"]\ntrust_level = \"trusted\"\n", out)
}

func TestSet_RootDottedTuiOtherKey(t *testing.T) {
	src := "tui.theme = \"x\"\n\n[features]\nhooks = true\n"
	out := mustSet(t, src, "model")
	require.NotContains(t, out, "[tui]", "a [tui] header would redefine the dotted table")
	require.Contains(t, out, "tui.status_line = [\"model\"]\n[features]")
}

func TestSet_BracketInComment(t *testing.T) {
	src := "[tui]\nstatus_line = [\"model\"] # see [docs]\n"
	require.Equal(t, "[tui]\nstatus_line = [\"git-branch\"]\n", mustSet(t, src, "git-branch"))
}

func TestSet_FakeHeaderInsideMultilineString(t *testing.T) {
	src := "desc = \"\"\"\n[tui]\nstatus_line = [\"nope\"]\n\"\"\"\n\n[tui]\ntheme = \"x\"\n"
	out := mustSet(t, src, "model")
	require.Contains(t, out, "\"\"\"\n[tui]\nstatus_line = [\"nope\"]\n\"\"\"", "string content untouched")
	require.Contains(t, out, "\n[tui]\nstatus_line = [\"model\"]\ntheme = \"x\"\n")
}

func TestSet_InlineTableRefused(t *testing.T) {
	_, err := SetStatusLine([]byte("tui = { theme = \"x\" }\n"), []string{"model"})
	require.ErrorContains(t, err, "inline table")
}

func TestSet_InvalidTOMLRefused(t *testing.T) {
	_, err := SetStatusLine([]byte("[tui\ntheme = \n"), []string{"model"})
	require.Error(t, err)
}

func TestRemoveAndGet(t *testing.T) {
	src := "[tui]\nstatus_line = [\n  \"model\",\n]\ntheme = \"x\"\n"
	out, ok, err := RemoveStatusLine([]byte(src))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "[tui]\ntheme = \"x\"\n", string(out))
	_, present, err := GetStatusLine(out)
	require.NoError(t, err)
	require.False(t, present)
	same, ok, err := RemoveStatusLine([]byte(userConfig))
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, userConfig, string(same))
}

// TestSet_HeaderWithBracketsInQuotedKey covers a header whose quoted key contains "["
// and "]" (defeating headerRe): a naive line-scanner would keep believing it is still
// inside the preceding [tui] table and could mistake the project's own "status_line"
// key for tui's. Neither Set nor Remove may touch that unrelated key.
func TestSet_HeaderWithBracketsInQuotedKey(t *testing.T) {
	src := "[tui]\ntheme = \"x\"\n\n[projects.\"/a/[b]\"]\nstatus_line = [\"nope\"]\n"

	// The real (only) tui.status_line is absent, so Remove must be a safe no-op: it
	// must not delete the projects table's unrelated "status_line" key.
	out, ok, err := RemoveStatusLine([]byte(src))
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, src, string(out))

	// Set must add a real tui.status_line without disturbing the projects key.
	got := mustSet(t, src, "model")
	require.Contains(t, got, "status_line = [\"nope\"]", "the project's own key must survive untouched")
	require.Contains(t, got, "[tui]\nstatus_line = [\"model\"]\ntheme = \"x\"\n")
}

// TestRemove_QuotedStatusLineKeyRefused covers a status_line key written with quotes
// ("status_line" = [...]), which GetStatusLine (real TOML parsing) sees as present but
// locate's plain-text regex does not recognize. Remove must refuse rather than silently
// leaving the key in place while reporting ok=false.
func TestRemove_QuotedStatusLineKeyRefused(t *testing.T) {
	src := "[tui]\n\"status_line\" = [\"model\"]\ntheme = \"x\"\n"
	out, ok, err := RemoveStatusLine([]byte(src))
	require.Error(t, err)
	require.False(t, ok)
	require.Equal(t, src, string(out))
}

func TestSetStatusLineWithComment(t *testing.T) {
	out, err := SetStatusLineWithComment([]byte("[tui]\ntheme = \"x\"\n"), []string{"model"}, ManagedComment)
	require.NoError(t, err)
	require.Equal(t, "[tui]\nstatus_line = [\"model\"] # agent-statusline\ntheme = \"x\"\n", string(out))

	ids, ok, err := GetStatusLine(out)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"model"}, ids)
	require.Equal(t, ManagedComment, StatusLineComment(out))
}

func TestSetStatusLineWithComment_EmptyCommentMatchesSetStatusLine(t *testing.T) {
	a, err := SetStatusLineWithComment([]byte("[tui]\ntheme = \"x\"\n"), []string{"model"}, "")
	require.NoError(t, err)
	b, err := SetStatusLine([]byte("[tui]\ntheme = \"x\"\n"), []string{"model"})
	require.NoError(t, err)
	require.Equal(t, string(b), string(a))
	require.Equal(t, "", StatusLineComment(a))
}

// StatusLineComment must recognise a status_line the user wrote themselves as unmarked (no
// ManagedComment), while GetStatusLine keeps parsing its ids normally.
func TestStatusLineComment_UnmarkedLineRoundTrips(t *testing.T) {
	src := []byte("[tui]\nstatus_line = [\"model\"]\ntheme = \"x\"\n")
	require.Equal(t, "", StatusLineComment(src))
	ids, ok, err := GetStatusLine(src)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"model"}, ids)
}

func TestStatusLineComment_NoStatusLine(t *testing.T) {
	require.Equal(t, "", StatusLineComment([]byte("[tui]\ntheme = \"x\"\n")))
	require.Equal(t, "", StatusLineComment([]byte("")))
}

// TestSet_NeighboringKeysSurvive guards against an over-wide splice: SetStatusLine's
// verification must compare the whole document (via sameIgnoringStatusLine), not just
// tui.status_line, so a bad splice that ate a neighbouring key would be caught.
func TestSet_NeighboringKeysSurvive(t *testing.T) {
	src := "[tui]\nstatus_line = [\n  \"model\",\n]\ntheme = \"x\"\nnotify = true\n"
	out := mustSet(t, src, "context-used")
	require.Contains(t, out, "theme = \"x\"")
	require.Contains(t, out, "notify = true")
}
