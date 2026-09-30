package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/engine"
)

const maxPayload = 4 << 20

func newRenderCmd() *cobra.Command {
	var harnessFlag string
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Read a harness payload on stdin and print the status line (called by Claude Code / Cursor)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxPayload))
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "agent-statusline: reading stdin: %v\n", err)
			}
			out := engine.Render(engine.Input{
				Payload: data, Harness: harnessFlag, Config: loadConfigLenient(cmd.ErrOrStderr()),
				Getenv: os.Getenv, NoColor: flagNoColor, Logf: logger(cmd.ErrOrStderr()),
			})
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil // a status line command must never fail
		},
	}
	cmd.Flags().StringVar(&harnessFlag, "harness", "auto", "payload format: claude | cursor | auto")
	return cmd
}
