package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/theme"
)

// PreviewFunc renders cfg for a harness.
type PreviewFunc func(cfg *config.Config, harness string) string

type InitResult struct {
	Config    *config.Config
	Harnesses []string
	Install   bool
}

var harnessLabels = map[string]string{"claude": "Claude Code", "cursor": "Cursor CLI", "codex": "Codex CLI"}

// RunInit is the first-run wizard: tools, layout, theme, segments per tool, then install.
func RunInit(base *config.Config, detected map[string]bool, preview PreviewFunc) (*InitResult, error) {
	cfg := base.Clone()
	result := &InitResult{}
	var options []huh.Option[string]
	for _, harness := range []string{"claude", "cursor", "codex"} {
		label := harnessLabels[harness]
		if !detected[harness] {
			label += " (not detected)"
		}
		options = append(options, huh.NewOption(label, harness).Selected(detected[harness]))
	}
	preset, themeName := "two-line", cfg.Theme
	err := huh.NewForm(
		huh.NewGroup(huh.NewMultiSelect[string]().Title("Which tools should agent-statusline manage?").
			Options(options...).Value(&result.Harnesses)),
		lookGroup(cfg, &preset, &themeName, preview, false),
	).Run()
	if err != nil {
		return nil, err
	}
	if err := applyLook(cfg, preset, themeName); err != nil {
		return nil, err
	}
	for _, harness := range result.Harnesses {
		if err := pickSegments(cfg, harness, preview); err != nil {
			return nil, err
		}
	}
	install := true
	if len(result.Harnesses) > 0 {
		if err := huh.NewConfirm().Title("Install into the selected tools now?").
			Description("Each settings file is backed up first.").Value(&install).Run(); err != nil {
			return nil, err
		}
	}
	result.Config, result.Install = cfg, install && len(result.Harnesses) > 0
	return result, nil
}

// RunEdit re-opens layout, theme and segment choices on an existing config.
func RunEdit(base *config.Config, preview PreviewFunc) (*config.Config, error) {
	cfg := base.Clone()
	preset, themeName := "keep", cfg.Theme
	var harnesses []string
	err := huh.NewForm(
		lookGroup(cfg, &preset, &themeName, preview, true),
		huh.NewGroup(huh.NewMultiSelect[string]().Title("Edit segments for").
			Options(huh.NewOption("Claude Code", "claude"), huh.NewOption("Cursor CLI", "cursor"), huh.NewOption("Codex CLI", "codex")).
			Value(&harnesses)),
	).Run()
	if err != nil {
		return nil, err
	}
	if err := applyLook(cfg, preset, themeName); err != nil {
		return nil, err
	}
	for _, harness := range harnesses {
		if err := pickSegments(cfg, harness, preview); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func lookGroup(cfg *config.Config, preset, themeName *string, preview PreviewFunc, allowKeep bool) *huh.Group {
	presets := huh.NewOptions(config.PresetNames...)
	if allowKeep {
		presets = append([]huh.Option[string]{huh.NewOption("keep current", "keep")}, presets...)
	}
	return huh.NewGroup(
		huh.NewSelect[string]().Title("Layout").Options(presets...).Value(preset),
		huh.NewSelect[string]().Title("Theme").Options(huh.NewOptions(theme.Names(cfg.Themes)...)...).Value(themeName),
		huh.NewNote().Title("Preview (Claude Code)").DescriptionFunc(func() string {
			clone := cfg.Clone()
			if err := applyLook(clone, *preset, *themeName); err != nil {
				return err.Error()
			}
			return preview(clone, "claude")
		}, []any{preset, themeName}),
	)
}

func applyLook(cfg *config.Config, preset, themeName string) error {
	if preset != "" && preset != "keep" {
		if err := config.ApplyPreset(cfg, preset); err != nil {
			return err
		}
	}
	cfg.Theme = themeName
	return nil
}

func pickSegments(cfg *config.Config, harness string, preview PreviewFunc) error {
	lines, maxLines := idsOf(cfg.LinesForName(harness)), 3
	if harness == "codex" {
		lines, maxLines = [][]string{cfg.Codex.Items}, 1
	}
	picker := NewPicker("Segments for "+harnessLabels[harness], Pool(harness), lines, maxLines, func(lines [][]string) string {
		clone := cfg.Clone()
		setLines(clone, harness, lines)
		return preview(clone, harness)
	})
	if _, err := tea.NewProgram(picker).Run(); err != nil {
		return err
	}
	if picker.Cancelled {
		return huh.ErrUserAborted
	}
	if picker.Done {
		setLines(cfg, harness, picker.Lines())
	}
	return nil
}

func idsOf(lines []config.Line) [][]string {
	out := make([][]string, 0, len(lines))
	for _, line := range lines {
		row := make([]string, 0, len(line))
		for _, ref := range line {
			row = append(row, ref.ID)
		}
		out = append(out, row)
	}
	return out
}

func setLines(cfg *config.Config, harness string, lines [][]string) {
	if harness == "codex" {
		cfg.Codex.Items = nil
		if len(lines) > 0 {
			cfg.Codex.Items = slices.Clone(lines[0])
		}
		return
	}
	target := &cfg.Claude.Lines
	if harness == "cursor" {
		target = &cfg.Cursor.Lines
	}
	old := map[string]config.SegRef{}
	for _, line := range *target {
		for _, ref := range line {
			old[ref.ID] = ref
		}
	}
	out := make([]config.Line, 0, len(lines))
	for _, row := range lines {
		line := make(config.Line, 0, len(row))
		for _, id := range row {
			if ref, ok := old[id]; ok {
				line = append(line, ref)
			} else {
				line = append(line, config.SegRef{ID: id})
			}
		}
		out = append(out, line)
	}
	*target = out
}
