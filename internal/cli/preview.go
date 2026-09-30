package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/engine"
)

func newPreviewCmd() *cobra.Command {
	var h, fixture, themeName string
	var width int
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Render your config against a sample payload (no live session needed)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := loadConfigLenient(cmd.ErrOrStderr())
			if h == "codex" {
				fmt.Fprint(cmd.OutOrStdout(), codexPreview(cfg))
				return nil
			}
			if themeName != "" {
				cfg = cfg.Clone()
				cfg.Theme = themeName
			}
			data, err := readFixture(fixture, h)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), engine.Render(engine.Input{
				Payload: data, Harness: h, Config: cfg, Getenv: os.Getenv, NoColor: flagNoColor,
				Now: engine.PreviewNow, Width: width, DisableCache: true, Run: engine.PreviewRunner,
				Logf: logger(cmd.ErrOrStderr()),
			}))
			return nil
		},
	}
	cmd.Flags().StringVar(&h, "harness", "claude", "claude | cursor | codex")
	cmd.Flags().StringVar(&fixture, "fixture", "", "bundled fixture (claude-full, claude-minimal, cursor-full, cursor-minimal) or a JSON file path")
	cmd.Flags().StringVar(&themeName, "theme", "", "override the configured theme")
	cmd.Flags().IntVar(&width, "width", 0, "fit lines to this many columns (0 = no fitting)")
	return cmd
}
