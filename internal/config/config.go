// Package config loads the single YAML file. Defaults live in default.yaml (embedded), so the
// commented file written by `config init-file` and the in-code defaults can never drift.
package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/joshi008/agent-statusline/internal/fsutil"
	"github.com/joshi008/agent-statusline/internal/model"
	"github.com/joshi008/agent-statusline/internal/theme"
)

//go:embed default.yaml
var defaultYAML []byte

type Config struct {
	Version  int                       `yaml:"version"`
	Theme    string                    `yaml:"theme"`
	Color    string                    `yaml:"color"`
	Style    StyleOverrides            `yaml:"style"`
	Segments map[string]map[string]any `yaml:"segments,omitempty"`
	Custom   []Custom                  `yaml:"custom,omitempty"`
	Claude   ClaudeConfig              `yaml:"claude"`
	Cursor   CursorConfig              `yaml:"cursor"`
	Codex    CodexConfig               `yaml:"codex"`
	Themes   map[string]theme.Spec     `yaml:"themes,omitempty"`
}

type StyleOverrides struct {
	Separator  *string     `yaml:"separator,omitempty"`
	Bar        BarOverride `yaml:"bar,omitempty"`
	Thresholds Thresholds  `yaml:"thresholds"`
	Countdown  *bool       `yaml:"countdown,omitempty"`
	Emoji      *bool       `yaml:"emoji,omitempty"`
}

type BarOverride struct {
	Width  *int    `yaml:"width,omitempty"`
	Glyphs *string `yaml:"glyphs,omitempty"`
}

type Thresholds struct {
	Warn float64 `yaml:"warn"`
	Crit float64 `yaml:"crit"`
}

// Custom is a user segment: exactly one of Run (shell command, cached for TTL) or Text.
type Custom struct {
	ID   string        `yaml:"id"`
	Run  string        `yaml:"run,omitempty"`
	Text string        `yaml:"text,omitempty"`
	TTL  time.Duration `yaml:"ttl,omitempty"`
}

type ClaudeConfig struct {
	RefreshInterval int            `yaml:"refresh_interval"`
	Lines           []Line         `yaml:"lines"`
	Subagents       SubagentConfig `yaml:"subagents"`
}

type SubagentConfig struct {
	Enabled bool   `yaml:"enabled"`
	Format  string `yaml:"format"`
}

type CursorConfig struct {
	UpdateIntervalMs int    `yaml:"update_interval_ms"`
	TimeoutMs        int    `yaml:"timeout_ms"`
	Lines            []Line `yaml:"lines"`
}

type CodexConfig struct {
	Items []string `yaml:"items"`
}

// Line is one status line row: an ordered list of segment references.
type Line []SegRef

// SegRef is a segment id, optionally with inline options: `model` or `{ id: context, style: pct }`.
type SegRef struct {
	ID   string
	Opts map[string]any
}

func (r *SegRef) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		r.ID, r.Opts = n.Value, nil
		return nil
	case yaml.MappingNode:
		var m map[string]any
		if err := n.Decode(&m); err != nil {
			return err
		}
		id, _ := m["id"].(string)
		if id == "" {
			return fmt.Errorf("line %d: segment map needs an id", n.Line)
		}
		delete(m, "id")
		r.ID, r.Opts = id, nil
		if len(m) > 0 {
			r.Opts = m
		}
		return nil
	}
	return fmt.Errorf("line %d: a segment must be an id or a map with an id", n.Line)
}

func (r SegRef) MarshalYAML() (any, error) {
	if len(r.Opts) == 0 {
		return r.ID, nil
	}
	m := map[string]any{"id": r.ID}
	for k, v := range r.Opts {
		m[k] = v
	}
	return m, nil
}

// DefaultYAML returns the commented default file.
func DefaultYAML() []byte { return bytes.Clone(defaultYAML) }

// Default returns a fresh copy of the built-in defaults.
func Default() *Config {
	c := &Config{}
	if err := yaml.Unmarshal(defaultYAML, c); err != nil {
		panic(fmt.Sprintf("config: embedded default.yaml is invalid: %v", err))
	}
	return c
}

// Parse decodes user YAML strictly (unknown keys are errors) over the defaults.
func Parse(data []byte) (*Config, error) {
	c := Default()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return c, nil
}

// Load reads path; a missing file means defaults.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// LoadLenient is Load for render paths. A file whose only problem is unknown keys is decoded
// again without KnownFields, so the rest of the user's settings still apply, and each unknown
// key comes back as a warning. Any other problem (bad YAML, a wrong type) is still an error.
func LoadLenient(path string) (*Config, []string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	c, err := Parse(b)
	if err == nil {
		return c, nil, nil
	}
	var te *yaml.TypeError
	if !errors.As(err, &te) || !allUnknownFields(te.Errors) {
		return nil, nil, err
	}
	c = Default()
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, nil, err
	}
	return c, te.Errors, nil
}

// allUnknownFields reports whether every strict-decode error is yaml.v3's unknown-key message
// ("line N: field X not found in type T").
func allUnknownFields(msgs []string) bool {
	for _, m := range msgs {
		if !strings.Contains(m, "field ") || !strings.Contains(m, " not found in type ") {
			return false
		}
	}
	return len(msgs) > 0
}

const saveHeader = "# Written by agent-statusline. Comments are not preserved;\n# `agent-statusline config init-file --force` restores the commented default.\n"

// Save writes c atomically. Comments in an existing file are lost (documented in the header).
func Save(path string, c *Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(path, append([]byte(saveHeader), b...), 0o644)
}

// DefaultPath is $XDG_CONFIG_HOME/agent-statusline/config.yaml or ~/.config/agent-statusline/config.yaml.
func DefaultPath(getenv func(string) string, home string) string {
	if x := getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "agent-statusline", "config.yaml")
	}
	return filepath.Join(home, ".config", "agent-statusline", "config.yaml")
}

// LinesFor returns the configured lines for a rendering harness.
func (c *Config) LinesFor(h model.Harness) []Line {
	if h == model.HarnessCursor {
		return c.Cursor.Lines
	}
	return c.Claude.Lines
}

// LinesForName is LinesFor keyed by "claude" / "cursor".
func (c *Config) LinesForName(h string) []Line { return c.LinesFor(model.Harness(h)) }

// SegmentOpts returns the `segments:` options for id (nil when none).
func (c *Config) SegmentOpts(id string) map[string]any { return c.Segments[id] }

// FindCustom looks up a custom segment by its bare id (without the cmd:/text: prefix).
func (c *Config) FindCustom(id string) (Custom, bool) {
	for _, cu := range c.Custom {
		if cu.ID == id {
			return cu, true
		}
	}
	return Custom{}, false
}

// Clone deep-copies via a YAML round trip (used by preview and the wizard).
func (c *Config) Clone() *Config {
	b, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	out := &Config{}
	if err := yaml.Unmarshal(b, out); err != nil {
		panic(err)
	}
	return out
}
