// Package doctor checks that everything agent-statusline depends on is wired and healthy.
package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/joshi008/agent-statusline/internal/codex"
	"github.com/joshi008/agent-statusline/internal/config"
	"github.com/joshi008/agent-statusline/internal/installer"
	"github.com/joshi008/agent-statusline/internal/segments"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

func (s Status) Symbol() string { return [...]string{"✓", "!", "✗"}[s] }

type Check struct {
	Name   string
	Status Status
	Detail string
}

type Deps struct {
	Getenv       func(string) string
	Home         string
	GOOS         string
	ConfigPath   string
	LookPath     func(string) (string, error)
	Stat         func(string) (fs.FileInfo, error)
	CodexVersion func() (string, error)
	RenderTime   func() time.Duration
}

func Failed(checks []Check) int {
	count := 0
	for _, check := range checks {
		if check.Status == Fail {
			count++
		}
	}
	return count
}

type adder func(name string, status Status, format string, args ...any)

func Run(d Deps) []Check {
	var checks []Check
	add := func(name string, status Status, format string, args ...any) {
		checks = append(checks, Check{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
	}
	if d.Stat == nil {
		d.Stat = os.Stat
	}

	if path, err := d.LookPath(installer.Marker); err == nil {
		add("binary", OK, "%s", path)
	} else {
		add("binary", Warn, "agent-statusline is not on PATH; `install` will point at the current executable")
	}
	checkConfig(d, add)
	checkJSON(d, add, "claude", installer.ClaudePath(d.Getenv, d.Home))
	checkJSON(d, add, "cursor", installer.CursorPath(d.Getenv, d.Home, d.GOOS))
	checkCodex(d, add, installer.CodexPath(d.Getenv, d.Home))

	if d.RenderTime != nil {
		renderTime := d.RenderTime()
		milliseconds := float64(renderTime.Microseconds()) / 1000
		if renderTime > 10*time.Millisecond {
			add("speed", Warn, "render takes %.1f ms (budget 10 ms)", milliseconds)
		} else {
			add("speed", OK, "render takes %.2f ms", milliseconds)
		}
	}
	switch strings.ToLower(d.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		add("colour", OK, "truecolor")
	default:
		add("colour", Warn, "COLORTERM=%q; colours fall back to 256 or 16 colours", d.Getenv("COLORTERM"))
	}
	terminal := d.Getenv("TERM_PROGRAM")
	switch terminal {
	case "iTerm.app", "WezTerm", "ghostty", "vscode":
		add("links", OK, "%s supports clickable PR links", terminal)
	default:
		if terminal == "" {
			terminal = "this terminal"
		}
		add("links", OK, "%s has no OSC 8 links, so PR numbers render as plain text (segments.pr.link: always forces them)", terminal)
	}
	return checks
}

func checkConfig(d Deps, add adder) {
	data, err := os.ReadFile(d.ConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		add("config", OK, "no file at %s; using defaults", d.ConfigPath)
		return
	}
	if err != nil {
		add("config", Fail, "%v", err)
		return
	}
	cfg, err := config.Parse(data)
	if err != nil {
		add("config", Fail, "%s: %v", d.ConfigPath, err)
		return
	}
	if errs := config.Validate(cfg, segments.Known); len(errs) > 0 {
		messages := make([]string, len(errs))
		for i, problem := range errs {
			messages[i] = problem.Error()
		}
		add("config", Fail, "%s", strings.Join(messages, "; "))
		return
	}
	add("config", OK, "%s", d.ConfigPath)
}

func firstToken(command string) string {
	command = strings.TrimSpace(command)
	var token strings.Builder
	var quote byte
	escaped := false
	for i := 0; i < len(command); i++ {
		char := command[i]
		if escaped {
			token.WriteByte(char)
			escaped = false
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else if char == '\\' && quote == '"' {
				escaped = true
			} else {
				token.WriteByte(char)
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '\\':
			escaped = true
		case ' ', '\t', '\r', '\n':
			return token.String()
		default:
			token.WriteByte(char)
		}
	}
	return token.String()
}

func checkJSON(d Deps, add adder, harness, path string) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		add(harness, Warn, "%s not found (is %s installed?)", path, harness)
		return
	}
	if err != nil {
		add(harness, Fail, "%v", err)
		return
	}
	if !gjson.ValidBytes(data) {
		add(harness, Fail, "%s is not valid JSON", path)
		return
	}
	command := gjson.GetBytes(data, "statusLine.command").String()
	switch {
	case command == "":
		add(harness, Warn, "no statusLine in %s (run `agent-statusline install %s`)", path, harness)
	case !strings.Contains(command, installer.Marker):
		add(harness, Warn, "statusLine runs %q, not agent-statusline (run `agent-statusline install %s`)", command, harness)
	default:
		if executable := firstToken(command); executable != "" {
			if _, err := d.Stat(executable); err != nil {
				add(harness, Fail, "statusLine points at %s, which does not exist (run `agent-statusline install %s` again)", executable, harness)
				break
			}
		}
		add(harness, OK, "%s", command)
	}
	switch harness {
	case "claude":
		if gjson.GetBytes(data, "disableAllHooks").Bool() {
			add(harness, Fail, "disableAllHooks is true, so Claude Code runs no status line")
		}
		if refresh := gjson.GetBytes(data, "statusLine.refreshInterval"); refresh.Exists() {
			switch value := refresh.Int(); {
			case value < 1:
				add(harness, Warn, "refreshInterval %d is below the minimum of 1 second", value)
			case value > 3600:
				add(harness, Warn, "refreshInterval %d is in seconds (%s), so rate-limit countdowns will go stale", value, time.Duration(value)*time.Second)
			}
		}
	case "cursor":
		if timeout := gjson.GetBytes(data, "statusLine.timeoutMs"); timeout.Exists() && timeout.Int() < 50 {
			add(harness, Warn, "timeoutMs %d is below Cursor's floor of 50", timeout.Int())
		}
	}
}

func checkCodex(d Deps, add adder, path string) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		add("codex", Warn, "%s not found (is Codex installed?)", path)
		return
	}
	if err != nil {
		add("codex", Fail, "%v", err)
		return
	}
	ids, ok, err := codex.GetStatusLine(data)
	switch {
	case err != nil:
		add("codex", Fail, "%s is not valid TOML: %v", path, err)
	case !ok:
		add("codex", Warn, "no tui.status_line in %s (run `agent-statusline install codex`)", path)
	default:
		var unknown []string
		for _, id := range ids {
			if !codex.IsItem(id) {
				unknown = append(unknown, id)
			}
		}
		switch {
		case len(unknown) > 0:
			add("codex", Fail, "unknown Codex items: %s", strings.Join(unknown, ", "))
		case strings.TrimSpace(codex.StatusLineComment(data)) != codex.ManagedComment:
			add("codex", Warn, "tui.status_line = %s was not set by agent-statusline (run `agent-statusline install codex`)", codex.FormatArray(ids))
		default:
			add("codex", OK, "status_line = %s", codex.FormatArray(ids))
		}
	}
	if d.CodexVersion == nil {
		return
	}
	if version, err := d.CodexVersion(); err == nil {
		fields := strings.Fields(version)
		if len(fields) > 0 && fields[len(fields)-1] != codex.VerifiedVersion {
			add("codex", Warn, "Codex %s installed; item ids were verified against %s", fields[len(fields)-1], codex.VerifiedVersion)
		}
	}
}
