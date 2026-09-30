package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit and build date",
		RunE: func(cmd *cobra.Command, _ []string) error {
			info, ok := debug.ReadBuildInfo()
			v, c, d := versionStrings(Version, Commit, Date, info, ok)
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "agent-statusline %s (%s, %s)\n", v, c, d)
			return err
		},
	}
}

// versionStrings returns the ldflags-stamped version, commit and date, or, for a build without
// them (Version == "dev", as with `go install`), the module version and VCS stamp Go embeds.
func versionStrings(version, commit, date string, info *debug.BuildInfo, ok bool) (string, string, string) {
	if version != "dev" || !ok || info == nil {
		return version, commit, date
	}
	if info.Main.Version != "" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch {
		case s.Key == "vcs.revision" && s.Value != "":
			commit = s.Value
		case s.Key == "vcs.time" && s.Value != "":
			date = s.Value
		}
	}
	return version, commit, date
}
