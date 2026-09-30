// Package cli wires the cobra commands.
package cli

import "github.com/spf13/cobra"

// Build metadata, overridden with -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Global flags shared by every command.
var (
	flagConfig  string
	flagNoColor bool
	flagVerbose bool
)

// NewRootCmd builds the command tree. Subcommands are added by later tasks.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "agent-statusline",
		Short:         "One status line for Claude Code, Cursor CLI and Codex",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&flagConfig, "config", "", "config file (default: $XDG_CONFIG_HOME/agent-statusline/config.yaml)")
	root.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable ANSI colours")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "log diagnostics to stderr")
	root.AddCommand(newVersionCmd(), newRenderCmd(), newRenderSubagentsCmd(), newPreviewCmd(), newThemesCmd(),
		newInstallCmd(), newUninstallCmd(), newConfigCmd(), newInitCmd(), newDoctorCmd())
	return root
}
