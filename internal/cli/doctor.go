package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/joshi008/agent-statusline/internal/cache"
	"github.com/joshi008/agent-statusline/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config, tool wiring, Codex item ids, speed and terminal support",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			checks := doctor.Run(doctor.Deps{
				Getenv: os.Getenv, Home: homeDir(), GOOS: runtime.GOOS, ConfigPath: configPath(),
				LookPath: exec.LookPath, Stat: os.Stat,
				CodexVersion: func() (string, error) {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					output, err := exec.CommandContext(ctx, "codex", "--version").Output()
					return strings.TrimSpace(string(output)), err
				},
				RenderTime: func() time.Duration {
					cfg := loadConfigLenient(io.Discard)
					const iterations = 20
					start := time.Now()
					for range iterations {
						previewString(cfg, "claude", 0, true)
					}
					return time.Since(start) / iterations
				},
			})
			for _, check := range checks {
				fmt.Fprintf(out, "%s %-7s %s\n", check.Status.Symbol(), check.Name, tildify(check.Detail))
			}
			if count, _ := cache.Prune(cache.DefaultDir(os.Getenv, homeDir()), 7*24*time.Hour, time.Now()); count > 0 {
				fmt.Fprintf(out, "✓ cache   pruned %d old entries\n", count)
			}
			if failed := doctor.Failed(checks); failed > 0 {
				return fmt.Errorf("%d check(s) failed", failed)
			}
			return nil
		},
	}
}
