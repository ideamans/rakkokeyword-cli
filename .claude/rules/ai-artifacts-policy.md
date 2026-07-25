# Generated artifacts — do not hand-edit

| Generated file | Source of truth |
| --- | --- |
| `internal/llmdocs/90-commands.md` | the cobra command definitions in `cmd/rakko/*.go`, rendered by the hidden `gen-llmdocs` command |

Hand-written and safe to edit:

- `internal/llmdocs/00-guide.md` — rules, auth, which command answers what
- `internal/llmdocs/10-metrics.md` — metric meanings, units, freshness
- `internal/llmdocs/20-schemas.md` — JSON output schemas per endpoint
- `internal/llmdocs/30-gotchas.md` — credits, limits, async behaviour, traps
- `plugins/rakkokeyword-cli/skills/*/SKILL.md`
- `context7.json`

Editing a generated file is always wrong: the next `go generate ./...`
overwrites it, and CI fails on the stale diff in the meantime. To improve the
catalog, improve the command's `Short` / `Long` / `Example` / flag usage strings
instead — the `--filter` path lists in the help text are generated from the
`FilterSpec` values in `internal/rakko/params.go`.

Regenerate with `/regen-ai`, or `go generate ./... && go test ./...`.
