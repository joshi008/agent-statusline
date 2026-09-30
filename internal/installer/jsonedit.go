package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"unicode"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/joshi008/agent-statusline/internal/codex"
)

// readJSONObject returns the file, "{}\n" when missing or empty, or an error when not an object.
func readJSONObject(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(bytes.TrimSpace(b)) == 0) {
		return []byte("{}\n"), nil
	}
	if err != nil {
		return nil, err
	}
	if !gjson.ValidBytes(b) || !gjson.ParseBytes(b).IsObject() {
		return nil, fmt.Errorf("%s is not valid JSON; fix it or move it aside, then retry", path)
	}
	return b, nil
}

func marshal(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// setKey replaces a top-level key's value in place, or appends it before the closing brace,
// leaving every other byte of the document untouched.
func setKey(src []byte, key string, value any) ([]byte, error) {
	return setKeyRaw(src, key, marshal(value))
}

// setKeyRaw is setKey with an already-encoded JSON value, so a raw value read back from a
// backup (via gjson's .Raw) can be restored verbatim instead of being re-marshaled.
func setKeyRaw(src []byte, key string, raw []byte) ([]byte, error) {
	if gjson.GetBytes(src, key).Exists() {
		return sjson.SetRawBytes(src, key, raw)
	}
	end := bytes.LastIndexByte(src, '}')
	if end < 0 {
		return nil, errors.New("settings file is not a JSON object")
	}
	body := bytes.TrimRightFunc(src[:end], unicode.IsSpace)
	var b bytes.Buffer
	b.Write(body)
	if len(body) > 0 && body[len(body)-1] != '{' {
		b.WriteByte(',')
	}
	b.WriteString("\n  ")
	b.Write(marshal(key))
	b.WriteString(": ")
	b.Write(raw)
	b.WriteString("\n")
	b.Write(src[end:])
	return b.Bytes(), nil
}

func isOurs(src []byte, key string) bool {
	return strings.Contains(gjson.GetBytes(src, key+".command").String(), Marker)
}

// keyOwned reports whether key is currently ours in doc, for the given harness. Codex has only
// one managed key ("status_line") and no "<key>.command" shape, so its ownership is recognised
// via the ManagedComment marker (StatusLineComment) instead of isOurs.
func keyOwned(harness string, doc []byte, key string) bool {
	if harness == "codex" {
		return strings.TrimSpace(codex.StatusLineComment(doc)) == codex.ManagedComment
	}
	return isOurs(doc, key)
}

// keepFunc builds the backup-protection function for a harness's managed keys (see
// pruneBackups). Given the backups (oldest first), it returns the set of backups pruning must
// keep: for each key, only the single backup uninstall would restore that key from — the one
// restoreSource picks. At most one backup per key is protected, so pruning stays bounded.
func keepFunc(harness string, keys ...string) func(backups []string) map[string]bool {
	return func(backups []string) map[string]bool {
		keep := map[string]bool{}
		for _, k := range keys {
			if p, _, ok := restoreSource(backups, harness, k); ok {
				keep[p] = true
			}
		}
		return keep
	}
}

// restoreSource returns the newest readable backup (backups are oldest first) in which key is
// not ours: the backup uninstall restores key from, and the one pruning protects for key.
func restoreSource(backups []string, harness, key string) (path string, doc []byte, ok bool) {
	for i := len(backups) - 1; i >= 0; i-- {
		b, err := os.ReadFile(backups[i])
		if err != nil || keyOwned(harness, b, key) {
			continue
		}
		return backups[i], b, true
	}
	return "", nil, false
}

func deleteIfOurs(src []byte, keys ...string) []byte {
	for _, k := range keys {
		if isOurs(src, k) {
			if out, err := sjson.DeleteBytes(src, k); err == nil {
				src = out
			}
		}
	}
	return src
}

// shellQuote quotes a path for Claude Code, which runs the command through a shell.
func shellQuote(p string) string {
	safe := true
	for _, r := range p {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("/._-+~", r)) {
			safe = false
			break
		}
	}
	if safe {
		return p
	}
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}
