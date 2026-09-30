package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/joshi008/agent-statusline/internal/codex"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/fsutil"
	"github.com/joshi008/agent-statusline/internal/segments"
)

// configSubcommands lets later tasks register subcommands without changing this constructor.
var configSubcommands []func() *cobra.Command

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Inspect, validate or create the config file"}
	cmd.AddCommand(newConfigPathCmd(), newConfigShowCmd(), newConfigValidateCmd(), newConfigInitFileCmd())
	for _, build := range configSubcommands {
		cmd.AddCommand(build())
	}
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use: "path", Short: "Print the config file path", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := configPath()
			if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintln(cmd.OutOrStdout(), path+" (not created yet)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show", Short: "Print the effective config (defaults merged with your file)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfigStrict()
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use: "validate", Short: "Check the config file and report every problem", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			path := configPath()
			data, err := os.ReadFile(path)
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(out, "✓ no config file at %s; built-in defaults are valid\n", path)
				return nil
			}
			if err != nil {
				return err
			}
			cfg, err := config.Parse(data)
			if err != nil {
				fmt.Fprintf(out, "✗ %s: %v\n", path, err)
				return errors.New("config is invalid")
			}
			errs := config.Validate(cfg, segments.Known)
			for _, problem := range errs {
				fmt.Fprintf(out, "✗ %v\n", problem)
			}
			if _, dropped := codex.Translate(cfg.Codex.Items); len(dropped) > 0 {
				fmt.Fprintf(out, "! codex: no item for %s (these are skipped for Codex)\n", strings.Join(dropped, ", "))
			}
			if len(errs) > 0 {
				return fmt.Errorf("%d problem(s) in %s", len(errs), path)
			}
			fmt.Fprintf(out, "✓ %s is valid\n", path)
			return nil
		},
	}
}

func newConfigInitFileCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use: "init-file", Short: "Write the commented default config file", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := configPath()
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists; use --force to overwrite it", path)
			}
			if err := fsutil.WriteAtomic(path, config.DefaultYAML(), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing file")
	return cmd
}
