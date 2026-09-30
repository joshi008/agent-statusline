# Contributing

Thanks for helping. `agent-statusline` is small on purpose. Changes that keep it fast, predictable and safe with people's settings files are the easiest to merge.

## Build and test

You need Go 1.25 or newer.

```sh
make build        # ./bin/agent-statusline
make test         # go test ./...
make bench        # render benchmark (budget: 10 ms)
go vet ./...
gofmt -l ./internal ./cmd   # must print nothing
```

Every change needs a test. Behaviour that users see is pinned by the golden files in `internal/engine/testdata/golden`. If you change rendered output on purpose, regenerate them and review the diff:

```sh
go test ./internal/engine -run TestGolden -update
git diff internal/engine/testdata/golden
```

Never run `install` or `uninstall` against your real tool settings while developing. Point them at a temporary home instead:

```sh
export HOME=$(mktemp -d) CLAUDE_CONFIG_DIR=$HOME/.claude CURSOR_CONFIG_DIR=$HOME/.cursor CODEX_HOME=$HOME/.codex
```

## Add a segment

1. Register it in `internal/segments` (`identity.go`, `location.go` or `meters.go`) with `register(Def{...})`, and add its id to `CatalogOrder`.
2. Set `Claude` / `Cursor` to where its data exists, and `Codex` to the matching Codex item id if there is one.
3. Return `nil` when the data is absent. Never render a made-up zero.
4. Add it to `DropOrder` in `internal/engine/layout.go` and to the segment table in `README.md`.
5. Test it in the segments package, as the existing tests do.

## Add a theme

Drop a YAML file into `internal/theme/builtin/`, add its name to `builtinOrder` in `internal/theme/theme.go`, and add a case to the theme tests. Themes are data only, so no rendering code should change.

## Pull requests

Keep one change per pull request. Describe what a user will see differently, and include the test that pins it.
