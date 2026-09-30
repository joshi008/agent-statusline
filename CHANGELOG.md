# Changelog

All notable changes are recorded here. The project follows [semantic versioning](https://semver.org); before v1.0, minor versions may change config keys or segment ids.

## [Unreleased] — v0.1.0

First release.

- `render` for Claude Code and Cursor CLI, and `render-subagents` for Claude Code's subagent rows.
- Codex CLI support through `[tui].status_line`, with ids verified against Codex 0.153.4 to 0.159.2.
- 26 built-in segments plus `cmd:<id>` and `text:<id>` custom segments.
- Themes as YAML data: `default`, `gradient`, `powerline`, and user themes via `extends`.
- One YAML config with `two-line`, `one-line` and `minimal` layout presets.
- `install` / `uninstall` with per-file backups, per-key ownership and in-place restore.
- `init` wizard and `config edit` picker with live preview; `config path | show | validate | init-file`.
- `preview`, `themes` and `doctor`.
- goreleaser config for darwin/linux (amd64/arm64) archives and a Homebrew formula.
