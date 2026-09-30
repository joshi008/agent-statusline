package codex

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var (
	headerRe    = regexp.MustCompile(`^\s*\[\s*([^\[\]]+?)\s*\]\s*(#.*)?$`)
	assignRe    = regexp.MustCompile(`^\s*[^\s\[#=][^=]*=`)
	tableKeyRe  = regexp.MustCompile(`^(\s*)status_line\s*=`)
	dottedKeyRe = regexp.MustCompile(`^(\s*)tui\s*\.\s*status_line\s*=`)
	rootTUIRe   = regexp.MustCompile(`^\s*tui\s*\.`)
	inlineTUIRe = regexp.MustCompile(`^\s*tui\s*=`)
)

type location struct {
	found         bool
	start, end    int // line range [start, end) of the existing assignment
	indent        string
	dotted        bool
	tuiHeader     int // line index of the [tui] header, -1 when absent
	firstHeader   int // line index of the first table header, len(lines) when none
	rootTUIDotted bool
	tuiInline     bool
}

// locate scans lines tracking the current table, skipping over multi-line values and strings.
func locate(lines []string) location {
	loc := location{tuiHeader: -1, firstHeader: len(lines)}
	table := ""
	markHeader := func(i int) {
		if loc.firstHeader == len(lines) {
			loc.firstHeader = i
		}
	}
	for i := 0; i < len(lines); {
		l := lines[i]
		if strings.HasPrefix(strings.TrimSpace(l), "[[") {
			table = "[[array]]"
			markHeader(i)
			i++
			continue
		}
		if m := headerRe.FindStringSubmatch(l); m != nil {
			table = strings.ReplaceAll(m[1], " ", "")
			markHeader(i)
			if table == "tui" && loc.tuiHeader < 0 {
				loc.tuiHeader = i
			}
			i++
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			// A header-shaped line we couldn't parse (e.g. a quoted key containing
			// "[" or "]", which defeats headerRe). Treat it as an unknown table so
			// keys below it are never mistaken for belonging to [tui].
			table = "\x00unknown-header\x00"
			markHeader(i)
			i++
			continue
		}
		if assignRe.MatchString(l) {
			end := valueEnd(lines, i, strings.Index(l, "=")+1)
			if table == "" {
				loc.rootTUIDotted = loc.rootTUIDotted || rootTUIRe.MatchString(l)
				loc.tuiInline = loc.tuiInline || inlineTUIRe.MatchString(l)
			}
			if !loc.found {
				if m := tableKeyRe.FindStringSubmatch(l); m != nil && table == "tui" {
					loc.found, loc.start, loc.end, loc.indent = true, i, end, m[1]
				} else if m := dottedKeyRe.FindStringSubmatch(l); m != nil && table == "" {
					loc.found, loc.start, loc.end, loc.indent, loc.dotted = true, i, end, m[1], true
				}
			}
			i = end
			continue
		}
		i++
	}
	return loc
}

// valueEnd returns the index after the last line of the value starting at lines[i][col:].
func valueEnd(lines []string, i, col int) int {
	depth, inStr := 0, ""
	for j := i; j < len(lines); j++ {
		l := lines[j]
		k := 0
		if j == i {
			k = col
		}
		for k < len(l) {
			rest := l[k:]
			switch {
			case inStr == "":
				switch {
				case strings.HasPrefix(rest, `"""`), strings.HasPrefix(rest, `'''`):
					inStr = rest[:3]
					k += 3
					continue
				case rest[0] == '"' || rest[0] == '\'':
					inStr = rest[:1]
				case rest[0] == '#':
					k = len(l)
					continue
				case rest[0] == '[' || rest[0] == '{':
					depth++
				case rest[0] == ']' || rest[0] == '}':
					depth--
				}
			case (inStr == `"` || inStr == `"""`) && rest[0] == '\\':
				k += 2
				continue
			case strings.HasPrefix(rest, inStr):
				k += len(inStr)
				inStr = ""
				continue
			}
			k++
		}
		if depth <= 0 && inStr == "" {
			return j + 1
		}
	}
	return len(lines)
}

func splice(lines []string, a, b int, insert ...string) []string {
	out := make([]string, 0, len(lines)-(b-a)+len(insert))
	out = append(out, lines[:a]...)
	out = append(out, insert...)
	return append(out, lines[b:]...)
}

// ManagedComment is the trailing comment SetStatusLineWithComment writes on the status_line
// line when the installer sets it, so Uninstall can recognise an entry as one we wrote
// (versus a status_line the user configured themselves) via StatusLineComment.
const ManagedComment = "agent-statusline"

// SetStatusLine sets [tui].status_line, touching nothing else, and verifies the result parses.
func SetStatusLine(src []byte, ids []string) ([]byte, error) {
	return SetStatusLineWithComment(src, ids, "")
}

// SetStatusLineWithComment is SetStatusLine, but appends " # "+comment to the status_line line
// when comment is non-empty (see ManagedComment / StatusLineComment).
func SetStatusLineWithComment(src []byte, ids []string, comment string) ([]byte, error) {
	if len(src) > 0 {
		if _, _, err := GetStatusLine(src); err != nil {
			return nil, fmt.Errorf("codex config is not valid TOML: %w", err)
		}
	}
	lines := strings.Split(string(src), "\n")
	loc := locate(lines)
	if loc.tuiInline {
		return nil, errors.New("codex config defines tui as an inline table; add status_line to it by hand")
	}
	value := FormatArray(ids)
	if comment != "" {
		value += " # " + comment
	}
	var candidates [][]string
	switch {
	case loc.found:
		key := "status_line"
		if loc.dotted {
			key = "tui.status_line"
		}
		candidates = append(candidates, splice(lines, loc.start, loc.end, loc.indent+key+" = "+value))
	case loc.tuiHeader >= 0:
		candidates = append(candidates, splice(lines, loc.tuiHeader+1, loc.tuiHeader+1, "status_line = "+value))
	case loc.rootTUIDotted:
		candidates = append(candidates, splice(lines, loc.firstHeader, loc.firstHeader, "tui.status_line = "+value))
	default:
		base := lines
		for len(base) > 0 && strings.TrimSpace(base[len(base)-1]) == "" {
			base = base[:len(base)-1]
		}
		c := slices.Clone(base)
		if len(c) > 0 {
			c = append(c, "")
		}
		candidates = append(candidates, append(c, "[tui]", "status_line = "+value, ""))
	}
	var lastErr error
	for _, c := range candidates {
		out := []byte(strings.Join(c, "\n"))
		got, ok, err := GetStatusLine(out)
		if err != nil || !ok || !slices.Equal(got, ids) {
			lastErr = fmt.Errorf("verification failed (got %v, present %v): %v", got, ok, err)
			continue
		}
		same, err := SameIgnoringStatusLine(src, out)
		if err != nil || !same {
			lastErr = fmt.Errorf("verification failed: something other than tui.status_line changed (same=%v): %v", same, err)
			continue
		}
		return out, nil
	}
	return nil, fmt.Errorf("codex config: could not set tui.status_line safely: %w", lastErr)
}

// RemoveStatusLine deletes the status_line assignment; ok is false when there was none.
// It refuses (returning an error and the untouched src) rather than risk touching the
// wrong key when it cannot be sure the edit is safe.
func RemoveStatusLine(src []byte) ([]byte, bool, error) {
	_, present, err := GetStatusLine(src)
	if err != nil {
		return src, false, err
	}
	if !present {
		return src, false, nil
	}
	lines := strings.Split(string(src), "\n")
	loc := locate(lines)
	if !loc.found {
		return src, false, errors.New("codex config: could not remove tui.status_line safely; edit it by hand")
	}
	out := []byte(strings.Join(splice(lines, loc.start, loc.end), "\n"))
	_, stillPresent, err := GetStatusLine(out)
	if err != nil || stillPresent {
		return src, false, errors.New("codex config: could not remove tui.status_line safely; edit it by hand")
	}
	if same, err := SameIgnoringStatusLine(src, out); err != nil || !same {
		return src, false, errors.New("codex config: could not remove tui.status_line safely; edit it by hand")
	}
	return out, true, nil
}

// SameIgnoringStatusLine reports whether a and b parse to the same TOML document once
// tui.status_line (and an emptied-out [tui] table) is disregarded in both.
func SameIgnoringStatusLine(a, b []byte) (bool, error) {
	da, err := decodeWithoutStatusLine(a)
	if err != nil {
		return false, err
	}
	db, err := decodeWithoutStatusLine(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(da, db), nil
}

func decodeWithoutStatusLine(src []byte) (map[string]any, error) {
	var doc map[string]any
	if err := toml.Unmarshal(src, &doc); err != nil {
		return nil, err
	}
	if tui, ok := doc["tui"].(map[string]any); ok {
		delete(tui, "status_line")
		if len(tui) == 0 {
			delete(doc, "tui")
		}
	}
	return doc, nil
}

// StatusLineComment returns the trailing "# ..." comment on the located tui.status_line
// assignment line (the "#" and surrounding space trimmed), or "" when there is no status_line,
// locate cannot pin down its assignment, or it has no trailing comment.
func StatusLineComment(src []byte) string {
	lines := strings.Split(string(src), "\n")
	loc := locate(lines)
	if !loc.found {
		return ""
	}
	// The comment we write always sits on the same physical line as the value itself,
	// which for our own writes (and the common single-line case) is the last line of the
	// (possibly multi-line) assignment.
	return trailingComment(lines[loc.end-1])
}

// trailingComment returns the text after the first '#' on line that is not inside a quoted
// string, or "" if there is none.
func trailingComment(line string) string {
	var q byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case q != 0:
			if c == '\\' && q == '"' {
				i++
				continue
			}
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			q = c
		case c == '#':
			return strings.TrimSpace(line[i+1:])
		}
	}
	return ""
}

// GetStatusLine parses src and returns [tui].status_line.
func GetStatusLine(src []byte) ([]string, bool, error) {
	var doc struct {
		TUI struct {
			StatusLine *[]string `toml:"status_line"`
		} `toml:"tui"`
	}
	if err := toml.Unmarshal(src, &doc); err != nil {
		return nil, false, err
	}
	if doc.TUI.StatusLine == nil {
		return nil, false, nil
	}
	return *doc.TUI.StatusLine, true, nil
}
