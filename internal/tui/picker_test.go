package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func press(p *Picker, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		_, cmd = p.Update(key(k))
	}
	return cmd
}

func pool() []Item {
	return []Item{{ID: "a", Available: true}, {ID: "b", Available: true}, {ID: "c", Available: false}, {ID: "d", Available: true}}
}

func TestPicker_InitialOrderAndLines(t *testing.T) {
	p := NewPicker("t", pool(), [][]string{{"b"}, {"d"}}, 3, nil)
	require.Equal(t, [][]string{{"b"}, {"d"}}, p.Lines())
	require.Equal(t, "b", p.items[0].ID)
	require.Equal(t, "d", p.items[1].ID)
}

func TestPicker_ToggleReorderMoveLine(t *testing.T) {
	p := NewPicker("t", pool(), [][]string{{"b"}, {"d"}}, 3, nil)
	press(p, "j", "j", " ")
	require.Equal(t, [][]string{{"b"}, {"d", "a"}}, p.Lines())
	press(p, "K")
	require.Equal(t, [][]string{{"b"}, {"a", "d"}}, p.Lines())
	press(p, "tab")
	require.Equal(t, [][]string{{"b"}, {"d"}, {"a"}}, p.Lines())
	press(p, "tab")
	require.Equal(t, [][]string{{"b", "a"}, {"d"}}, p.Lines())
	press(p, "k", " ")
	require.Equal(t, [][]string{{"a"}, {"d"}}, p.Lines())
}

func TestPicker_UnavailableCannotBeEnabled(t *testing.T) {
	p := NewPicker("t", pool(), nil, 3, nil)
	press(p, "j", "j", " ")
	require.Empty(t, p.Lines())
}

func TestPicker_KeepsCustomIDsAndClamps(t *testing.T) {
	p := NewPicker("t", pool(), [][]string{{"cmd:k8s", "a"}}, 3, nil)
	require.Equal(t, [][]string{{"cmd:k8s", "a"}}, p.Lines())
	press(p, "k", "k", "K")
	require.Equal(t, 0, p.cursor)
	press(p, "j", "j", "j", "j", "j", "j", "J")
	require.Equal(t, len(p.items)-1, p.cursor)
}

func TestPicker_EnterAndEsc(t *testing.T) {
	p := NewPicker("t", pool(), nil, 3, nil)
	require.NotNil(t, press(p, "enter"))
	require.True(t, p.Done)
	p = NewPicker("t", pool(), nil, 3, nil)
	press(p, "esc")
	require.True(t, p.Cancelled)
}

func TestPicker_View(t *testing.T) {
	p := NewPicker("Segments for claude", pool(), [][]string{{"b"}}, 3, func([][]string) string { return "PREVIEW" })
	view := p.View()
	require.Contains(t, view, "Segments for claude")
	require.Contains(t, view, "[x] L1 b")
	require.Contains(t, view, " -     c")
	require.Contains(t, view, "PREVIEW")
}

func TestPool(t *testing.T) {
	find := func(items []Item, id string) Item {
		for _, item := range items {
			if item.ID == id {
				return item
			}
		}
		t.Fatalf("missing %s", id)
		return Item{}
	}
	require.True(t, find(Pool("cursor"), "max_mode").Available)
	require.False(t, find(Pool("cursor"), "cost").Available)
	require.True(t, find(Pool("codex"), "cost").Available)
	require.False(t, find(Pool("codex"), "git_status").Available)
}
