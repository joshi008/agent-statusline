// Package fixtures embeds realistic harness payloads used by tests and `preview`.
package fixtures

import (
	"embed"
	"fmt"
)

//go:embed *.json
var fs embed.FS

// Names lists the available fixtures in display order.
var Names = []string{"claude-full", "claude-minimal", "cursor-full", "cursor-minimal"}

// Read returns the fixture bytes. It panics on an unknown name; callers pass literals.
func Read(name string) []byte {
	b, err := fs.ReadFile(name + ".json")
	if err != nil {
		panic(fmt.Sprintf("fixtures: unknown fixture %q", name))
	}
	return b
}
