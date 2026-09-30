package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/installer"
)

var harnessNames = []string{"claude", "cursor", "codex"}

var restartHint = map[string]string{
	"claude": "takes effect on the next status line refresh",
	"cursor": "restart cursor-agent to apply",
	"codex":  "restart Codex to apply",
}

func harnessPath(h string) string {
	switch h {
	case "claude":
		return installer.ClaudePath(os.Getenv, homeDir())
	case "cursor":
		return installer.CursorPath(os.Getenv, homeDir(), runtime.GOOS)
	}
	return installer.CodexPath(os.Getenv, homeDir())
}

func tildify(s string) string {
	if h := homeDir(); h != "" {
		return strings.ReplaceAll(s, h+"/", "~/")
	}
	return s
}

// targets expands "all" to detected tools; an explicit name is always honoured.
func targets(arg string, w io.Writer) ([]string, error) {
	if arg == "" || arg == "all" {
		found := installer.Detect(homeDir(), exec.LookPath)
		var out []string
		for _, h := range harnessNames {
			if found[h] {
				out = append(out, h)
			} else {
				fmt.Fprintf(w, "- %-6s skipped (not installed)\n", h)
			}
		}
		return out, nil
	}
	for _, h := range harnessNames {
		if h == arg {
			return []string{h}, nil
		}
	}
	return nil, fmt.Errorf("unknown tool %q (want claude, cursor, codex or all)", arg)
}

func report(w io.Writer, res installer.Result, err error, verb string) {
	switch {
	case err != nil:
		fmt.Fprintf(w, "✗ %-6s %s: %v\n", res.Harness, tildify(res.Path), err)
		return
	case !res.Changed:
		fmt.Fprintf(w, "= %-6s %s already up to date\n", res.Harness, tildify(res.Path))
	default:
		fmt.Fprintf(w, "✓ %-6s %s %s (%s)\n", res.Harness, tildify(res.Path), verb, restartHint[res.Harness])
		if res.Backup != "" {
			fmt.Fprintf(w, "         backup: %s\n", tildify(res.Backup))
		}
	}
	for _, warn := range res.Warnings {
		fmt.Fprintf(w, "  ! %s\n", warn)
	}
	if len(res.Dropped) > 0 {
		fmt.Fprintf(w, "  ! Codex has no item for: %s (skipped)\n", strings.Join(res.Dropped, ", "))
	}
}

func runInstallList(w io.Writer, harnesses []string, cfg *config.Config, bin string) error {
	var failed []string
	for _, h := range harnesses {
		var res installer.Result
		var err error
		now := time.Now()
		switch h {
		case "claude":
			res, err = installer.InstallClaude(harnessPath(h), bin, cfg, now)
		case "cursor":
			res, err = installer.InstallCursor(harnessPath(h), bin, cfg, now)
		case "codex":
			res, err = installer.InstallCodex(harnessPath(h), cfg, now)
		}
		report(w, res, err, "updated")
		if err != nil {
			failed = append(failed, h)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("install failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func runUninstallList(w io.Writer, harnesses []string) error {
	var failed []string
	for _, h := range harnesses {
		res, err := installer.Uninstall(h, harnessPath(h), time.Now())
		report(w, res, err, "restored")
		if err != nil {
			failed = append(failed, h)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("uninstall failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "install [claude|cursor|codex|all]",
		Short:     "Point Claude Code, Cursor CLI and Codex at agent-statusline (backs up each file)",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: append(harnessNames, "all"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigStrict()
			if err != nil {
				return err
			}
			bin, err := installer.StableBinary()
			if err != nil {
				return err
			}
			if err := installer.CheckBinary(bin); err != nil {
				return err
			}
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			hs, err := targets(arg, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return runInstallList(cmd.OutOrStdout(), hs, cfg, bin)
		},
	}
}

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "uninstall [claude|cursor|codex|all]",
		Short:     "Remove agent-statusline from the tools (restores your previous status line when possible)",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: append(harnessNames, "all"),
		RunE: func(cmd *cobra.Command, args []string) error {
			hs := harnessNames
			if len(args) == 1 && args[0] != "all" {
				var err error
				if hs, err = targets(args[0], cmd.OutOrStdout()); err != nil {
					return err
				}
			}
			return runUninstallList(cmd.OutOrStdout(), hs)
		},
	}
}
