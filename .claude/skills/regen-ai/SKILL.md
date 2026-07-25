---
name: regen-ai
description: Regenerate the embedded LLM reference and verify the result. Use after changing commands, flags, filter paths, help text, JSON output, or the hand-written reference chapters.
allowed-tools: Bash(go generate:*) Bash(go test:*) Bash(go build:*) Bash(git status:*) Bash(git diff:*) Read
---

# regen-ai

Bring `internal/llmdocs/` back in line with the code.

1. `git status --short` — note what is already dirty.
2. `go generate ./...` — rewrites `90-commands.md`.
3. `go build ./... && go test ./...`.
4. Report which commands, flags or filter paths changed.

Then check by hand what generation cannot see:

- **JSON output changed?** `20-schemas.md` needs a matching edit. The generator
  only sees the command tree, not the shape of what commands print.
- **Cost changed?** It appears in five places: the `Cost:` line in the command's
  `Long`, the `credits` field of its `call`, both READMEs and `30-gotchas.md`.
  The distributed SKILL.md holds no cost table on purpose — only the three
  extremes — so check it just for those.
- **New endpoint?** `00-guide.md` has a "which command answers which question"
  table that generation does not touch.

This skill is Claude Code-local; it is not part of the distributed plugin.
