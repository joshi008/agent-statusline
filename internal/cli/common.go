package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/joshi008/agent-statusline/internal/codex"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/engine"
	"github.com/joshi008/agent-statusline/internal/fixtures"
)

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func configPath() string {
	if flagConfig != "" {
		return flagConfig
	}
	return config.DefaultPath(os.Getenv, homeDir())
}

// loadConfigLenient is for render paths: unknown keys are warned about on stderr and skipped;
// any other problem is reported and defaults are used.
func loadConfigLenient(stderr io.Writer) *config.Config {
	c, warnings, err := config.LoadLenient(configPath())
	if err != nil {
		fmt.Fprintf(stderr, "agent-statusline: config %s: %v (using defaults)\n", configPath(), err)
		return config.Default()
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "agent-statusline: config %s: %s (ignored)\n", configPath(), w)
	}
	return c
}

// loadConfigStrict is for commands that change things: a broken config is an error.
func loadConfigStrict() (*config.Config, error) {
	c, err := config.Load(configPath())
	if err != nil {
		return nil, fmt.Errorf("config %s: %w (run `agent-statusline config validate`)", configPath(), err)
	}
	return c, nil
}

func logger(w io.Writer) func(string, ...any) {
	if !flagVerbose {
		return nil
	}
	return func(f string, a ...any) { fmt.Fprintf(w, "agent-statusline: "+f+"\n", a...) }
}

// readFixture accepts a bundled fixture name ("claude-full") or a file path; "" = "<harness>-full".
func readFixture(nameOrPath, harness string) ([]byte, error) {
	if nameOrPath == "" {
		nameOrPath = harness + "-full"
	}
	if slices.Contains(fixtures.Names, nameOrPath) {
		return fixtures.Read(nameOrPath), nil
	}
	return os.ReadFile(nameOrPath)
}

// previewString renders cfg against the bundled fixture for harness with canned git/gh data.
func previewString(cfg *config.Config, harness string, width int, noColor bool) string {
	if harness == "codex" {
		return codexPreview(cfg)
	}
	data, err := readFixture("", harness)
	if err != nil {
		return err.Error()
	}
	return engine.Render(engine.Input{Payload: data, Harness: harness, Config: cfg, Getenv: os.Getenv,
		NoColor: noColor, Now: engine.PreviewNow, Width: width, DisableCache: true, Run: engine.PreviewRunner})
}

// codexPreview shows the TOML we would write; Codex renders the items itself.
func codexPreview(cfg *config.Config) string {
	ids, dropped := codex.Translate(cfg.Codex.Items)
	s := "[tui]\nstatus_line = " + codex.FormatArray(ids) + "\n"
	if len(dropped) > 0 {
		s += "# not available in Codex: " + strings.Join(dropped, ", ") + "\n"
	}
	return s
}

func writeFile(path, content string) error { return os.WriteFile(path, []byte(content), 0o644) }
