// Package installer wires agent-statusline into each tool's config file, surgically and with
// backups, and removes it again (restoring the user's previous status line when possible).
package installer

import (
	"os"
	"path/filepath"
)

// Marker identifies entries we own.
const Marker = "agent-statusline"

type Result struct {
	Harness  string
	Path     string
	Backup   string
	Changed  bool
	Dropped  []string
	Warnings []string
}

func ClaudePath(getenv func(string) string, home string) string {
	if d := getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "settings.json")
	}
	return filepath.Join(home, ".claude", "settings.json")
}

func CursorPath(getenv func(string) string, home, goos string) string {
	if d := getenv("CURSOR_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "cli-config.json")
	}
	if goos == "linux" {
		if x := getenv("XDG_CONFIG_HOME"); x != "" {
			return filepath.Join(x, "cursor", "cli-config.json")
		}
	}
	return filepath.Join(home, ".cursor", "cli-config.json")
}

func CodexPath(getenv func(string) string, home string) string {
	if d := getenv("CODEX_HOME"); d != "" {
		return filepath.Join(d, "config.toml")
	}
	return filepath.Join(home, ".codex", "config.toml")
}

// Detect reports which tools look installed (binary on PATH or config dir present).
func Detect(home string, lookPath func(string) (string, error)) map[string]bool {
	has := func(dir string, bins ...string) bool {
		for _, b := range bins {
			if _, err := lookPath(b); err == nil {
				return true
			}
		}
		_, err := os.Stat(filepath.Join(home, dir))
		return err == nil
	}
	return map[string]bool{
		"claude": has(".claude", "claude"),
		"cursor": has(".cursor", "cursor-agent"),
		"codex":  has(".codex", "codex"),
	}
}
