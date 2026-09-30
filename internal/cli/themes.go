package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/theme"
)

func newThemesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "themes",
		Short: "List themes with a one-line preview of each",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := loadConfigLenient(cmd.ErrOrStderr())
			for _, name := range theme.Names(cfg.Themes) {
				c := cfg.Clone()
				c.Theme = name
				c.Claude.Lines = []config.Line{{{ID: "model"}, {ID: "effort"}, {ID: "branch"}, {ID: "context"}, {ID: "cost"}, {ID: "limit_5h"}}}
				fmt.Fprintf(cmd.OutOrStdout(), "%-10s %s", name, previewString(c, "claude", 0, flagNoColor))
			}
			return nil
		},
	}
}
