package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/engine"
)

func newRenderSubagentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "render-subagents",
		Short: "Render Claude Code subagent rows (called via subagentStatusLine)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, _ := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxPayload))
			cfg := loadConfigLenient(cmd.ErrOrStderr())
			lvl := engine.ColorLevel(cfg.Color, flagNoColor, os.Getenv)
			fmt.Fprint(cmd.OutOrStdout(), engine.RenderSubagents(data, cfg, lvl, time.Now()))
			return nil
		},
	}
}
