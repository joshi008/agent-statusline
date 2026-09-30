// Package render holds ANSI painting, bars and width-aware text helpers. No terminal detection
// beyond environment variables: stdout is always a pipe when a harness runs us.
package render

import "strings"

type Level int

const (
	LevelNone Level = iota
	LevelBasic
	Level256
	LevelTrue
)

// DetectLevel picks the colour depth from env. NO_COLOR wins.
func DetectLevel(env func(string) string) Level {
	if env("NO_COLOR") != "" {
		return LevelNone
	}
	switch strings.ToLower(env("COLORTERM")) {
	case "truecolor", "24bit":
		return LevelTrue
	}
	if strings.Contains(env("TERM"), "256color") {
		return Level256
	}
	return LevelBasic
}

// ParseLevel maps a config value ("auto" handled by caller) to a Level.
func ParseLevel(s string) (Level, bool) {
	switch s {
	case "none":
		return LevelNone, true
	case "basic":
		return LevelBasic, true
	case "256":
		return Level256, true
	case "truecolor":
		return LevelTrue, true
	}
	return LevelNone, false
}
