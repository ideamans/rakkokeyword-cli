---
paths:
  - "cmd/rakkokeyword/*.go"
  - "internal/rakko/params.go"
  - "internal/output/*.go"
  - "internal/llmdocs/0*.md"
  - "internal/llmdocs/1*.md"
  - "internal/llmdocs/2*.md"
  - "internal/llmdocs/3*.md"
---

# You just touched the source of the embedded LLM reference

If you changed a command, a flag, a filter path or a help string, run
`/regen-ai` before finishing so `internal/llmdocs/90-commands.md` matches. CI
regenerates it and fails on a dirty tree.

Two things generation cannot do for you:

- **JSON output changed** — a field name, a type, an items path — then
  `internal/llmdocs/20-schemas.md` needs a matching edit by hand. Agents parse
  against that chapter, so a mismatch makes them misread data silently rather
  than fail.
- **Credit cost changed** — update the `Cost:` line in the command's `Long`, the
  `credits` field of its `call`, both README cost tables and the table in
  `30-gotchas.md`. A stale cost makes an agent quote the user a wrong price.
  `skills/rakkokeyword-usage/SKILL.md` deliberately carries no cost table — only the
  three extremes (other-keywords 22.5, the 15-credit search-volume minimum, the
  25× `--seo-difficulty` multiplier) — so touch it only when one of those moves.

Do not edit `90-commands.md` directly.
