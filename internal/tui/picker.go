// Package tui holds the interactive setup: a huh-based wizard and a Bubble Tea segment picker.
// Nothing here is imported by the render path.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshi008/agent-statusline/internal/segments"
)

type Item struct {
	ID        string
	Desc      string
	Available bool
}

// Pool lists every built-in segment, marking those that have data on harness.
func Pool(harness string) []Item {
	var items []Item
	for _, def := range segments.All() {
		items = append(items, Item{ID: def.ID, Desc: def.Desc, Available: def.Available(harness)})
	}
	return items
}

type Picker struct {
	Title     string
	MaxLines  int
	Preview   func(lines [][]string) string
	Done      bool
	Cancelled bool

	items  []Item
	line   map[string]int
	cursor int
}

// NewPicker orders enabled segments first, line by line in configured order, then the rest.
func NewPicker(title string, pool []Item, lines [][]string, maxLines int, preview func([][]string) string) *Picker {
	if maxLines < 1 {
		maxLines = 1
	}
	picker := &Picker{Title: title, MaxLines: maxLines, Preview: preview, line: map[string]int{}}
	byID := map[string]Item{}
	for _, item := range pool {
		byID[item.ID] = item
	}
	placed := map[string]bool{}
	for lineIndex, row := range lines {
		for _, id := range row {
			if placed[id] {
				continue
			}
			item, ok := byID[id]
			if !ok {
				item = Item{ID: id, Desc: "custom", Available: true}
			}
			line := lineIndex
			if line >= maxLines {
				line = maxLines - 1
			}
			picker.items = append(picker.items, item)
			picker.line[id] = line
			placed[id] = true
		}
	}
	for _, item := range pool {
		if !placed[item.ID] {
			picker.items = append(picker.items, item)
			picker.line[item.ID] = -1
		}
	}
	return picker
}

func (p *Picker) Init() tea.Cmd { return nil }

func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok || len(p.items) == 0 {
		return p, nil
	}
	switch key.String() {
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}
	case "K", "shift+up":
		p.swap(p.cursor - 1)
	case "J", "shift+down":
		p.swap(p.cursor + 1)
	case " ", "space", "x":
		p.toggle()
	case "tab":
		if id := p.items[p.cursor].ID; p.line[id] >= 0 {
			p.line[id] = (p.line[id] + 1) % p.MaxLines
		}
	case "enter":
		p.Done = true
		return p, tea.Quit
	case "esc", "q", "ctrl+c":
		p.Cancelled = true
		return p, tea.Quit
	}
	return p, nil
}

func (p *Picker) swap(other int) {
	if other < 0 || other >= len(p.items) {
		return
	}
	p.items[p.cursor], p.items[other] = p.items[other], p.items[p.cursor]
	p.cursor = other
}

func (p *Picker) toggle() {
	item := p.items[p.cursor]
	if !item.Available {
		return
	}
	if p.line[item.ID] >= 0 {
		p.line[item.ID] = -1
		return
	}
	p.line[item.ID] = 0
	for i := p.cursor - 1; i >= 0; i-- {
		if line := p.line[p.items[i].ID]; line >= 0 {
			p.line[item.ID] = line
			break
		}
	}
}

// Lines returns enabled ids per line, dropping empty lines.
func (p *Picker) Lines() [][]string {
	var out [][]string
	for line := 0; line < p.MaxLines; line++ {
		var ids []string
		for _, item := range p.items {
			if p.line[item.ID] == line {
				ids = append(ids, item.ID)
			}
		}
		if len(ids) > 0 {
			out = append(out, ids)
		}
	}
	return out
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

func (p *Picker) View() string {
	var out strings.Builder
	out.WriteString(titleStyle.Render(p.Title) + "\n\n")
	for index, item := range p.items {
		cursor := "  "
		if index == p.cursor {
			cursor = cursorStyle.Render("> ")
		}
		box, tag := "[ ]", "  "
		if line := p.line[item.ID]; line >= 0 {
			box, tag = "[x]", fmt.Sprintf("L%d", line+1)
		}
		if !item.Available {
			box = " - "
		}
		row := fmt.Sprintf("%s %s %-12s", box, tag, item.ID)
		if !item.Available {
			row = dimStyle.Render(row)
		}
		out.WriteString(cursor + row + " " + dimStyle.Render(item.Desc) + "\n")
	}
	out.WriteString("\n" + dimStyle.Render("↑/↓ move · space toggle · K/J reorder · tab next line · enter save · esc cancel") + "\n")
	if p.Preview != nil {
		out.WriteString("\n" + p.Preview(p.Lines()))
	}
	return out.String()
}
