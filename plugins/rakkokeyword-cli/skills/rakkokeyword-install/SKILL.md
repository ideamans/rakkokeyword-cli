---
name: rakkokeyword-install
description: Make the rakkokeyword command available, installing it only if it is missing. Use when another skill reports that `rakkokeyword` is not on PATH, or when the user asks to install, update or upgrade the ideamans rakkokeyword CLI. Prefers an already-installed binary, then the latest GitHub release, then a build from source with go install.
license: MIT
compatibility: Requires curl (or wget) and tar to install from a release, or a Go toolchain for the source fallback. Standalone — does not need rakkokeyword to be present already. Installs from the public repository github.com/ideamans/rakkokeyword-cli, so no GitHub authentication is needed.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(go:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# rakkokeyword-install

Make the `rakkokeyword` command usable, doing the least work that achieves it.

## Route 1 — an existing installation on PATH

```bash
command -v rakkokeyword && rakkokeyword --version && rakkokeyword llm | head -1
```

If that resolves, **use it and stop here** — do not check for a newer release,
it costs an API call and the user did not ask for an upgrade.

Two things invalidate the hit. `rakkokeyword` is a short name, so the first line of
`rakkokeyword llm` must read `# rakkokeyword — reference for AI agents`; if something else
owns the name, say so and use an explicit path rather than shadowing theirs. And
if `llm` is not a known command at all, the binary predates the embedded
reference — continue to route 2 to upgrade it.

## Route 2 — the latest GitHub release

The repository is public, so no authentication is needed.

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/rakkokeyword-cli/releases/latest \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)   # e.g. v0.1.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')             # darwin | linux
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64  # amd64 | arm64
curl -fsSL -o /tmp/rakkokeyword.tar.gz \
  "https://github.com/ideamans/rakkokeyword-cli/releases/download/${VERSION}/rakkokeyword_${VERSION#v}_${OS}_${ARCH}.tar.gz"
```

**The archive is named after the binary, not the repository** — `rakkokeyword`,
with no `-cli` suffix: `rakkokeyword_<version-without-v>_<os>_<arch>.tar.gz`.
Windows ships a `.zip`. If the download 404s, list the actual assets on the
release page rather than retrying variations.

```bash
tar -xzf /tmp/rakkokeyword.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/rakkokeyword ~/.local/bin/ && chmod +x ~/.local/bin/rakkokeyword
```

Prefer the first writable directory already on PATH — `~/.local/bin`, then
`/usr/local/bin`. Two things not to do on your own initiative:

- If nothing on PATH is writable, leave the binary in `/tmp`, print the exact
  `sudo mv` command and let the user run it. Do not run `sudo` yourself.
- If `~/.local/bin` is not on PATH, give the user the line to add to their shell
  profile. Do not edit the profile for them.

## Route 3 — build from source

Last resort: needs a Go toolchain and compiles rather than downloads. Note the
`/cmd/rakkokeyword` suffix — installing the module root would not build anything.

```bash
go install github.com/ideamans/rakkokeyword-cli/cmd/rakkokeyword@latest
```

The binary lands in `$(go env GOPATH)/bin`.

## Verify, then say what is still needed

```bash
rakkokeyword --version
rakkokeyword metadata languages | head -3   # free, needs no API key: proves it reaches the API
```

Report which route was taken, the version and the install path.

`rakkokeyword` cannot do keyword research without an API key from a rakkokeyword
STANDARD plan or above (up to 5 keys per account, issued in the account
settings):

```bash
export RAKKOKEYWORD_API_KEY=your-key   # preferred; leaves nothing on disk
rakkokeyword auth set-api-key your-key        # or store it in the config file
```

Everything beyond `metadata` spends credits from the user's account — mention
that before the first real query, and see the `rakkokeyword-usage` skill for costs.
