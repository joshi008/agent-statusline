package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/fsutil"
	"github.com/joshi008/agent-statusline/internal/installer"
	"github.com/joshi008/agent-statusline/internal/tui"
)

func init() { configSubcommands = append(configSubcommands, newConfigEditCmd) }

func interactive(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func tuiPreview(cfg *config.Config, harness string) string {
	width := 0
	if columns, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && columns > 8 {
		width = columns - 4
	}
	return previewString(cfg, harness, width, flagNoColor)
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive first-run setup: pick tools, layout, theme and segments, then install",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, path := cmd.OutOrStdout(), configPath()
			if !interactive(cmd) {
				if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
					if err := fsutil.WriteAtomic(path, config.DefaultYAML(), 0o644); err != nil {
						return err
					}
					fmt.Fprintf(out, "Wrote %s\n", path)
				} else {
					fmt.Fprintf(out, "Config already exists at %s\n", path)
				}
				fmt.Fprintln(out, "Not a terminal, so skipping the wizard. Next: agent-statusline preview, then agent-statusline install")
				return nil
			}
			base, err := loadConfigStrict()
			if err != nil {
				return err
			}
			result, err := tui.RunInit(base, installer.Detect(homeDir(), exec.LookPath), tuiPreview)
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Fprintln(out, "Cancelled; nothing was written.")
				return nil
			}
			if err != nil {
				return err
			}
			if err := config.Save(path, result.Config); err != nil {
				return err
			}
			fmt.Fprintf(out, "Saved %s\n", path)
			if !result.Install {
				fmt.Fprintln(out, "Run `agent-statusline install` when you are ready.")
				return nil
			}
			binary, err := installer.StableBinary()
			if err != nil {
				return err
			}
			if err := installer.CheckBinary(binary); err != nil {
				return err
			}
			return runInstallList(out, result.Harnesses, result.Config, binary)
		},
	}
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Interactively change layout, theme and segments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !interactive(cmd) {
				return fmt.Errorf("config edit needs an interactive terminal; edit %s directly", configPath())
			}
			base, err := loadConfigStrict()
			if err != nil {
				return err
			}
			cfg, err := tui.RunEdit(base, tuiPreview)
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled; nothing was written.")
				return nil
			}
			if err != nil {
				return err
			}
			if err := config.Save(configPath(), cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s (installed tools pick it up on their next refresh; run `install codex` again for Codex)\n", configPath())
			return nil
		},
	}
}
