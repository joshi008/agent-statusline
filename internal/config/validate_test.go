package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var knownIDs = map[string]bool{"model": true, "context": true, "session": true, "effort": true, "thinking": true,
	"fast": true, "dir": true, "branch": true, "git_status": true, "pr": true, "worktree": true, "agent": true,
	"cost": true, "limit_5h": true, "limit_7d": true, "spend": true, "cache": true, "max_mode": true, "autorun": true, "vim": true}

func known(id string) bool { return knownIDs[id] }

func TestValidate_DefaultIsValid(t *testing.T) {
	require.Empty(t, Validate(Default(), known))
}

func TestValidate_Errors(t *testing.T) {
	cases := map[string]string{
		"version: 2\n":     "version",
		"color: rainbow\n": "color",
		"theme: nope\n":    "theme",
		"style:\n  thresholds: { warn: 90, crit: 80 }\n":                  "thresholds",
		"style:\n  bar: { glyphs: \"#\" }\n":                              "glyphs",
		"claude:\n  lines: [[model, bogus]]\n":                            `unknown segment "bogus"`,
		"cursor:\n  lines: [[cmd:k8s]]\n":                                 `unknown custom segment "k8s"`,
		"custom: [{ id: k8s, text: x }]\nclaude:\n  lines: [[cmd:k8s]]\n": "is a text segment",
		"custom: [{ id: a, run: x, text: y }]\n":                          "exactly one of run or text",
		"custom: [{ id: a, run: x }, { id: a, run: y }]\n":                "duplicate",
		"custom: [{ run: x }]\n":                                          "id is required",
		"cursor:\n  update_interval_ms: 100\n":                            "minimum is 300",
		"cursor:\n  timeout_ms: 10\n":                                     "timeout_ms",
		"claude:\n  refresh_interval: -1\n":                               "refresh_interval",
	}
	for src, want := range cases {
		c, err := Parse([]byte(src))
		require.NoError(t, err, src)
		errs := Validate(c, known)
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		require.Contains(t, strings.Join(msgs, "\n"), want, src)
	}
}
