// Package theme turns theme data (embedded built-ins plus user YAML) into resolved styles.
// Themes are data, not code: adding a built-in means dropping a YAML file in builtin/.
package theme

import (
	"embed"
	"fmt"
	"path"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/joshi008/agent-statusline/internal/render"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// builtinOrder is the display order; every name must have builtin/<name>.yaml.
var builtinOrder = []string{"default", "gradient", "powerline"}

// Spec is the YAML shape of a theme. Nil/empty fields inherit from Extends.
type Spec struct {
	Extends   string            `yaml:"extends,omitempty"`
	Colors    map[string]string `yaml:"colors,omitempty"`
	Separator *string           `yaml:"separator,omitempty"`
	BarGlyphs *string           `yaml:"bar_glyphs,omitempty"`
	BarWidth  *int              `yaml:"bar_width,omitempty"`
	Gradient  *bool             `yaml:"gradient,omitempty"`
	Emoji     *bool             `yaml:"emoji,omitempty"`
	Caps      []string          `yaml:"caps,omitempty"`
	Icons     map[string]string `yaml:"icons,omitempty"`
}

// Theme is a fully resolved theme.
type Theme struct {
	Name      string
	Roles     map[string]render.Style
	Separator string
	BarGlyphs [2]rune
	BarWidth  int
	Gradient  bool
	Emoji     bool
	Caps      [2]string
	Icons     map[string]string
}

// Style returns the style for a role; unknown roles are the zero (unstyled) style.
func (t *Theme) Style(role string) render.Style { return t.Roles[role] }

// Builtins parses the embedded theme files. It panics on a broken embed (covered by tests).
func Builtins() map[string]Spec {
	out := make(map[string]Spec, len(builtinOrder))
	for _, name := range builtinOrder {
		b, err := builtinFS.ReadFile(path.Join("builtin", name+".yaml"))
		if err != nil {
			panic(err)
		}
		var s Spec
		if err := yaml.Unmarshal(b, &s); err != nil {
			panic(fmt.Sprintf("theme %s: %v", name, err))
		}
		out[name] = s
	}
	return out
}

// Names lists built-ins first (fixed order), then user-only themes sorted.
func Names(user map[string]Spec) []string {
	names := slices.Clone(builtinOrder)
	var extra []string
	for n := range user {
		if !slices.Contains(names, n) {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	return append(names, extra...)
}

// Resolve walks the extends chain (user themes shadow built-ins) and applies it root-first.
func Resolve(name string, user map[string]Spec) (*Theme, error) {
	builtins := Builtins()
	var chain []Spec
	seen := map[string]bool{}
	for cur := name; cur != ""; {
		if seen[cur] {
			return nil, fmt.Errorf("theme %q: extends cycle at %q", name, cur)
		}
		seen[cur] = true
		s, ok := user[cur]
		if !ok {
			s, ok = builtins[cur]
		}
		if !ok {
			return nil, fmt.Errorf("unknown theme %q", cur)
		}
		chain = append(chain, s)
		cur = s.Extends
	}
	t := &Theme{Name: name, Roles: map[string]render.Style{}, Separator: " · ",
		BarGlyphs: [2]rune{'▓', '░'}, BarWidth: 10, Icons: map[string]string{}}
	for i := len(chain) - 1; i >= 0; i-- {
		if err := t.apply(chain[i]); err != nil {
			return nil, fmt.Errorf("theme %q: %w", name, err)
		}
	}
	return t, nil
}

func (t *Theme) apply(s Spec) error {
	for role, v := range s.Colors {
		st, err := render.ParseStyle(v)
		if err != nil {
			return fmt.Errorf("colour %q: %w", role, err)
		}
		t.Roles[role] = st
	}
	if s.Separator != nil {
		t.Separator = *s.Separator
	}
	if s.BarGlyphs != nil {
		g, err := ParseGlyphs(*s.BarGlyphs)
		if err != nil {
			return err
		}
		t.BarGlyphs = g
	}
	if s.BarWidth != nil {
		if *s.BarWidth < 1 || *s.BarWidth > 50 {
			return fmt.Errorf("bar_width must be 1..50, got %d", *s.BarWidth)
		}
		t.BarWidth = *s.BarWidth
	}
	if s.Gradient != nil {
		t.Gradient = *s.Gradient
	}
	if s.Emoji != nil {
		t.Emoji = *s.Emoji
	}
	if s.Caps != nil {
		if len(s.Caps) != 2 {
			return fmt.Errorf("caps must have exactly two entries")
		}
		t.Caps = [2]string{s.Caps[0], s.Caps[1]}
	}
	for k, v := range s.Icons {
		t.Icons[k] = v
	}
	return nil
}

// ParseGlyphs requires exactly two runes: filled, empty.
func ParseGlyphs(s string) ([2]rune, error) {
	r := []rune(s)
	if len(r) != 2 {
		return [2]rune{}, fmt.Errorf("bar glyphs must be exactly two characters, got %q", s)
	}
	return [2]rune{r[0], r[1]}, nil
}
