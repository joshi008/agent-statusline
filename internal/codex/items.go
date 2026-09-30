// Package codex maps our segments onto Codex CLI's built-in footer items and edits
// ~/.codex/config.toml. Codex has no command-backed status line, so this is all we can do.
package codex

import (
	"slices"
	"strconv"
	"strings"

	"github.com/joshi008/agent-statusline/internal/segments"
)

// VerifiedVersion is the Codex release whose binary the item list was extracted from.
const VerifiedVersion = "0.159.2"

// KnownItems are the tui.status_line identifiers accepted by Codex 0.153.4 through 0.159.2 (checked against both binaries).
var KnownItems = []string{
	"model", "model-with-reasoning", "reasoning", "task-progress", "project-name", "current-dir",
	"run-state", "thread-title", "git-branch", "context-remaining", "context-used", "five-hour-limit",
	"weekly-limit", "thread-credits", "estimated-thread-cost", "codex-version", "used-tokens",
	"total-input-tokens", "total-output-tokens", "thread-id", "fast-mode",
}

func IsItem(id string) bool { return slices.Contains(KnownItems, id) }

// Translate maps config items (segment ids or raw Codex ids) to Codex ids.
func Translate(items []string) (ids, dropped []string) {
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for i := 0; i < len(items); i++ {
		it := items[i]
		if it == "model" && i+1 < len(items) && items[i+1] == "effort" {
			add("model-with-reasoning")
			i++
			continue
		}
		if d, ok := segments.Lookup(it); ok && d.Codex != "" {
			add(d.Codex)
			continue
		}
		if IsItem(it) {
			add(it)
			continue
		}
		dropped = append(dropped, it)
	}
	return ids, dropped
}

// FormatArray renders a TOML inline array of strings.
func FormatArray(ids []string) string {
	q := make([]string, len(ids))
	for i, id := range ids {
		q[i] = strconv.Quote(id)
	}
	return "[" + strings.Join(q, ", ") + "]"
}
