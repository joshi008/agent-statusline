package installer

import (
	"strings"
	"testing"
	"time"

	"github.com/joshi008/agent-statusline/internal/codex"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestUninstall_RestoresPreviousStatusLine(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	res, err := Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	require.Equal(t, claudeWithUserStatusLine, read(t, p), "the user's own status line is back")
}

func TestUninstall_KeepsLaterEditsWhenBackupDiffers(t *testing.T) {
	p := write(t, "settings.json", claudeSettings)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	edited := strings.Replace(read(t, p), `"effortLevel": "high"`, `"effortLevel": "max"`, 1)
	require.NoError(t, writeFileForTest(p, edited))

	_, err = Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)
	out := read(t, p)
	require.False(t, gjson.Get(out, "statusLine").Exists())
	require.False(t, gjson.Get(out, "subagentStatusLine").Exists())
	require.Equal(t, "max", gjson.Get(out, "effortLevel").String())
}

func TestUninstall_LeavesForeignStatusLine(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	res, err := Uninstall("claude", p, t0)
	require.NoError(t, err)
	require.False(t, res.Changed)
	require.Equal(t, claudeWithUserStatusLine, read(t, p))
}

func TestUninstall_CursorAndCodexRoundTrip(t *testing.T) {
	cur := write(t, "cli-config.json", `{"version": 1}`+"\n")
	_, err := InstallCursor(cur, bin, config.Default(), t0)
	require.NoError(t, err)
	_, err = Uninstall("cursor", cur, t0.Add(time.Second))
	require.NoError(t, err)
	// As above: the statusLine key is deleted and (here) nothing is restored in its place
	// since the backup never had one, but deleting a key that was appended can leave
	// formatting (e.g. a trailing newline before "}") different from the original bytes;
	// JSONEq checks the value, which must be exactly the original.
	require.JSONEq(t, `{"version": 1}`, read(t, cur))

	cx := write(t, "config.toml", codexConfig)
	_, err = InstallCodex(cx, config.Default(), t0)
	require.NoError(t, err)
	_, err = Uninstall("codex", cx, t0.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, codexConfig, read(t, cx))
}

func TestUninstall_MissingFileIsNoop(t *testing.T) {
	res, err := Uninstall("claude", "/nonexistent/settings.json", t0)
	require.NoError(t, err)
	require.False(t, res.Changed)
	_, err = Uninstall("vim", "/x", t0)
	require.Error(t, err)
}

// CRITICAL 1 regression: once the user has replaced our entry with their own, a later uninstall
// must never reach past it into an old backup and restore something stale over it.
func TestUninstall_ForeignStatusLineSurvivesEvenAfterWeOwnedItBefore(t *testing.T) {
	p := write(t, "settings.json", `{"a": 1}`+"\n")
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	_, err = Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)

	mine := `{"a": 1, "statusLine": {"command": "my.sh"}}` + "\n"
	require.NoError(t, writeFileForTest(p, mine))

	res, err := Uninstall("claude", p, t0.Add(2*time.Second))
	require.NoError(t, err)
	require.False(t, res.Changed)
	require.Equal(t, mine, read(t, p), "the user's own statusLine must survive byte-for-byte")
}

// CRITICAL 2 regression: a codex status_line with no ManagedComment marker was never installed
// by us and must never be removed.
func TestUninstall_CodexLeavesAStatusLineWeNeverInstalled(t *testing.T) {
	src := "[tui]\nstatus_line = [\"model\"]\n"
	p := write(t, "config.toml", src)
	res, err := Uninstall("codex", p, t0)
	require.NoError(t, err)
	require.False(t, res.Changed)
	require.Equal(t, src, read(t, p))
}

// IMPORTANT 1 regression: only the status line keys are restored from the backup; an unrelated
// edit made after install (here, to theme) must survive uninstall.
func TestUninstall_RestoresStatusLineKeepsUnrelatedEdit(t *testing.T) {
	p := write(t, "settings.json", claudeWithUserStatusLine)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	edited := strings.Replace(read(t, p), `"theme": "dark"`, `"theme": "light"`, 1)
	require.NoError(t, writeFileForTest(p, edited))

	res, err := Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	require.Equal(t, "bash /Users/me/.claude/statusline-command.sh", gjson.Get(out, "statusLine.command").String())
	require.EqualValues(t, 30000, gjson.Get(out, "statusLine.refreshInterval").Int())
	require.False(t, gjson.Get(out, "subagentStatusLine").Exists())
	require.Equal(t, "light", gjson.Get(out, "theme").String())
}

// Codex round trip when the file already had its own status_line before we installed: after
// uninstall the user's ids are back, our marker comment is gone, and other keys are untouched.
func TestUninstall_CodexRestoresUsersOwnPreviousStatusLine(t *testing.T) {
	src := "model = \"gpt-5.6-sol\"\n\n[tui]\nstatus_line = [\"model\"]\ntheme = \"x\"\n"
	p := write(t, "config.toml", src)
	_, err := InstallCodex(p, config.Default(), t0)
	require.NoError(t, err)

	res, err := Uninstall("codex", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	ids, ok, err := codex.GetStatusLine([]byte(out))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"model"}, ids)
	require.Equal(t, "", codex.StatusLineComment([]byte(out)))
	require.Contains(t, out, "theme = \"x\"")
	require.Contains(t, out, "model = \"gpt-5.6-sol\"")
}

// Fix round 2, IMPORTANT: uninstall must restore key k only when k is still ours in the current
// file — never a key the user has since replaced or removed themselves, even if an old backup
// still has a (now stale) value for it.

func TestUninstall_KeepsUsersReplacementSubagentStatusLine(t *testing.T) {
	original := `{"statusLine": {"command": "old.sh"}, "subagentStatusLine": {"command": "oldsub.sh"}}` + "\n"
	p := write(t, "settings.json", original)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)

	edited, err := sjson.Set(read(t, p), "subagentStatusLine.command", "newsub.sh")
	require.NoError(t, err)
	require.NoError(t, writeFileForTest(p, edited))

	res, err := Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	require.Equal(t, "old.sh", gjson.Get(out, "statusLine.command").String(), "our own statusLine is uninstalled and the foreign one restored")
	require.Equal(t, "newsub.sh", gjson.Get(out, "subagentStatusLine.command").String(), "the user's replacement must survive, not the stale backup value")
}

func TestUninstall_DoesNotResurrectAKeyTheUserDeleted(t *testing.T) {
	original := `{"statusLine": {"command": "old.sh"}, "subagentStatusLine": {"command": "oldsub.sh"}}` + "\n"
	p := write(t, "settings.json", original)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)

	edited, err := sjson.Delete(read(t, p), "subagentStatusLine")
	require.NoError(t, err)
	require.NoError(t, writeFileForTest(p, edited))

	res, err := Uninstall("claude", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	require.Equal(t, "old.sh", gjson.Get(out, "statusLine.command").String())
	require.False(t, gjson.Get(out, "subagentStatusLine").Exists(), "a key the user deleted must not be resurrected from an old backup")
}

// Fix round 3, controller ruling: a backup must never be skipped wholesale just because SOME
// key in it happens to be ours — each key picks its own newest non-owning backup independently.
// Here backup1 (mixed: statusLine is the user's X.sh, subagentStatusLine is still ours) must not
// be skipped entirely when restoring statusLine (which would wrongly fall through to the older,
// genuinely-foreign backup0's OLD.sh), and must correctly be skipped when restoring
// subagentStatusLine (whose value there is still ours, not a real foreign value).
func TestUninstall_PicksBackupPerKeyIndependently(t *testing.T) {
	p := write(t, "settings.json", `{"a": 1}`+"\n")

	// backup0: a genuinely foreign, older snapshot for both keys.
	older := `{"statusLine": {"command": "OLD.sh"}, "subagentStatusLine": {"command": "OLDSUB.sh"}}` + "\n"
	require.NoError(t, writeFileForTest(p, older))
	_, err := Backup(p, t0, keepFunc("claude", "statusLine", "subagentStatusLine"))
	require.NoError(t, err)

	// backup1: mixed — the user's own statusLine (X.sh) alongside our own subagentStatusLine.
	mixed, err := sjson.Set(older, "statusLine.command", "X.sh")
	require.NoError(t, err)
	mixed, err = sjson.Set(mixed, "subagentStatusLine.command", bin+" render-subagents")
	require.NoError(t, err)
	require.NoError(t, writeFileForTest(p, mixed))
	_, err = Backup(p, t0.Add(time.Second), keepFunc("claude", "statusLine", "subagentStatusLine"))
	require.NoError(t, err)

	// current file: as if we had just reinstalled over the mixed state, so both keys are ours.
	current, err := sjson.Set(mixed, "statusLine.command", bin+" render --harness claude")
	require.NoError(t, err)
	require.NoError(t, writeFileForTest(p, current))

	res, err := Uninstall("claude", p, t0.Add(2*time.Second))
	require.NoError(t, err)
	require.True(t, res.Changed)
	out := read(t, p)
	require.Equal(t, "X.sh", gjson.Get(out, "statusLine.command").String(),
		"the newer foreign statusLine must be restored, not skipped because the same backup's subagentStatusLine is ours")
	require.Equal(t, "OLDSUB.sh", gjson.Get(out, "subagentStatusLine.command").String(),
		"must fall through the mixed backup (still ours there) to the genuinely older foreign value")
}

// Final review, IMPORTANT 1: turning subagent rows off on a reinstall must hand the user's own
// subagentStatusLine back, or it is lost for good (uninstall later skips a key that isn't ours).
func TestInstallClaude_DisablingSubagentsRestoresUsersSubagentStatusLine(t *testing.T) {
	original := `{"statusLine": {"command": "old.sh"}, "subagentStatusLine": {"type": "command", "command": "mysub.sh"}}` + "\n"
	p := write(t, "settings.json", original)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)

	cfg := config.Default()
	cfg.Claude.Subagents.Enabled = false
	res, err := InstallClaude(p, bin, cfg, t0.Add(time.Second))
	require.NoError(t, err)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "restored your previous subagentStatusLine")
	require.Equal(t, "mysub.sh", gjson.Get(read(t, p), "subagentStatusLine.command").String())

	_, err = Uninstall("claude", p, t0.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, original, read(t, p), "the user's file comes back byte for byte")
}

// With no earlier value of the user's own, disabling subagent rows just removes ours.
func TestInstallClaude_DisablingSubagentsDeletesOursWhenNothingToRestore(t *testing.T) {
	p := write(t, "settings.json", claudeSettings)
	_, err := InstallClaude(p, bin, config.Default(), t0)
	require.NoError(t, err)
	cfg := config.Default()
	cfg.Claude.Subagents.Enabled = false
	_, err = InstallClaude(p, bin, cfg, t0.Add(time.Second))
	require.NoError(t, err)
	require.False(t, gjson.Get(read(t, p), "subagentStatusLine").Exists())
}

// Final review, MINOR A: the user's own top-level dotted tui.status_line comes back where it
// was, rather than as a new [tui] table appended to the end of the file.
func TestUninstall_CodexRestoresDottedStatusLineInPlace(t *testing.T) {
	src := "model = \"gpt-5.6-sol\"\ntui.status_line = [\"model\"]\n\n[profiles.x]\nmodel = \"y\"\n"
	p := write(t, "config.toml", src)
	_, err := InstallCodex(p, config.Default(), t0)
	require.NoError(t, err)
	require.NotEqual(t, src, read(t, p))

	_, err = Uninstall("codex", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, src, read(t, p))
}

// The same for a [tui] table where status_line is the only key: no empty [tui] is left behind
// and nothing moves.
func TestUninstall_CodexRestoresTableStatusLineInPlace(t *testing.T) {
	src := "[tui]\nstatus_line = [\"model\"]\n\n[profiles.x]\nmodel = \"y\"\n"
	p := write(t, "config.toml", src)
	_, err := InstallCodex(p, config.Default(), t0)
	require.NoError(t, err)
	_, err = Uninstall("codex", p, t0.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, src, read(t, p))
}
