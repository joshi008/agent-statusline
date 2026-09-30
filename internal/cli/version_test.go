package cli

import (
	"bytes"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	Version, Commit, Date = "1.2.3", "abc", "2026-09-29"
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	require.NoError(t, root.Execute())
	require.Equal(t, "agent-statusline 1.2.3 (abc, 2026-09-29)\n", out.String())
}

// Final review, MINOR D: a `go install` build (no ldflags) reports the module version and the
// VCS stamp from the embedded build info instead of "dev (none, unknown)".
func TestVersionStrings_FallsBackToBuildInfo(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.4.0"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "2847e72abc"}, {Key: "vcs.time", Value: "2026-09-29T10:00:00Z"},
	}}
	v, c, d := versionStrings("dev", "none", "unknown", info, true)
	require.Equal(t, []string{"v0.4.0", "2847e72abc", "2026-09-29T10:00:00Z"}, []string{v, c, d})

	v, c, d = versionStrings("1.2.3", "abc", "2026-09-29", info, true)
	require.Equal(t, []string{"1.2.3", "abc", "2026-09-29"}, []string{v, c, d}, "ldflags values win")

	noVCS := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	v, c, d = versionStrings("dev", "none", "unknown", noVCS, true)
	require.Equal(t, []string{"(devel)", "none", "unknown"}, []string{v, c, d})

	v, c, d = versionStrings("dev", "none", "unknown", nil, false)
	require.Equal(t, []string{"dev", "none", "unknown"}, []string{v, c, d})
}
