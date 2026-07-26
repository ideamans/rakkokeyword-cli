# rakkokeyword-cli

A command-line client for the [ラッコキーワード (rakkokeyword) API](https://api.rakkokeyword.com/api-docs.json):
keyword research, SEO metrics, live Google rankings, competitor analysis and
content briefs — mostly for the Japanese search market.

Every API operation is covered, output is `table` / `json` / `jsonl` / `csv`,
and the complete agent-facing reference is embedded in the binary behind
`rakkokeyword llm`.

日本語版は [README_ja.md](README_ja.md)。

```bash
rakkokeyword suggest-keywords ラッコ -n 5
```

```
consumed 1.5 credit(s)
keyword=ラッコ  totalCount=864  returnedCount=5
   keyword        suggestClass   metrics.searchVolume   metrics.seoDifficulty   metrics.cpc
------------------+------------+----------------------+-----------------------+-------------
  ラッコ            ＋                          90500                      35             0
  らっこ            ＋                          90500                      32             0
  ラッコキーワード   ＋                          49500                      40          6.43
  ラッコ キーワード  ＋α                         49500                      40          6.43
  ラッコ 水族館      ＋                          18100                      36          0.03
```

## Install

```bash
go install github.com/ideamans/rakkokeyword-cli/cmd/rakkokeyword@latest
```

Or download a binary from the [releases page](https://github.com/ideamans/rakkokeyword-cli/releases).
The archives are named after the binary (`rakkokeyword`), not the repository:
`rakkokeyword_<version>_<os>_<arch>.tar.gz`.

## Authentication

An API key comes from a rakkokeyword STANDARD plan or above (up to 5 keys per
account).

```bash
export RAKKOKEYWORD_API_KEY=your-key   # preferred: nothing is stored on disk
rakkokeyword auth set-api-key your-key        # or persist it in the config file
rakkokeyword auth status                      # shows the source, never the key
```

Resolution order: `--api-key` → `RAKKOKEYWORD_API_KEY` → `RAKKO_API_KEY` →
config file (`~/.config/rakkokeyword-cli/config.json`).

`rakkokeyword metadata locations` and `rakkokeyword metadata languages` work without a key.

## Credits

Calls are billed to the account behind the key, so every command states its
cost in `--help`, prints what it consumed on stderr, and can be priced without
sending anything:

```bash
rakkokeyword other-keywords ラッコ --dry-run
```

```json
{
  "body": { "keyword": "ラッコ" },
  "cost": "22.5 credits per request",
  "method": "POST",
  "url": "https://api.rakkokeyword.com/v1/other-keywords"
}
```

| Command | Cost |
| --- | --- |
| `suggest-keywords`, `related-keywords`, `site-search` | 1.5 |
| `question-search`, `headline`, `co-occurrence` | 3 |
| `ranking-keywords`, `influx-keywords`, `influx-pages`, `competitive`, `content-search` | 4.5 |
| `other-keywords` | 22.5 |
| `bulk-site-research` | 0.45 per URL, min 4.5 |
| `search-volume register` | 0.03 per keyword (+0.75 with `--seo-difficulty`), min 15 |
| `search-rank register` | 0.9 per keyword for ranks 1–30, +0.3 per extra 10 ranks |
| `metadata`, `histories`, `status`, `results` | free |

## Commands

### Keyword discovery

```bash
rakkokeyword suggest-keywords ラッコ --modes google,bing --increase-keyword   # search suggestions
rakkokeyword related-keywords ラッコ --match-type phraseMatch -n 5000         # bulk keyword harvest
rakkokeyword other-keywords ラッコ                                            # LSI + People Also Ask
rakkokeyword question-search ラッコ -n 200                                    # question phrasings
rakkokeyword ranking-keywords ラッコ --search-top 10 --search-range 20        # same-intent keywords
```

### Fresh metrics and rankings (asynchronous)

```bash
rakkokeyword search-volume register --keywords-file keywords.txt --wait
rakkokeyword search-volume histories
rakkokeyword search-volume status 1234567
rakkokeyword search-volume results 1234567 -n 500 -f csv

rakkokeyword search-rank register ラッコ --url https://example.com/ --depth 100 --device mobile --wait
rakkokeyword search-rank results 01HQZX… --with-aggregation -f json
```

`register --wait` performs register → poll → results in one invocation. Without
it, the printed `requestId` can be picked up later — jobs keep running.

### Sites and competitors

```bash
rakkokeyword influx-keywords --target https://example.com/ --match-type sub_domain
rakkokeyword influx-pages --target https://example.com/ -n 50
rakkokeyword competitive https://example.com/
rakkokeyword bulk-site-research --urls-file sites.txt
rakkokeyword content-search ラッコ --search-target title
rakkokeyword site-search --filter keyword.includes=ラッコ
```

### Content briefs

```bash
rakkokeyword headline ラッコ                       # h1–h6 of the ranking pages
rakkokeyword co-occurrence ラッコ --details=false   # vocabulary of the ranking pages
```

### Escape hatch

```bash
rakkokeyword raw POST /v1/suggest-keywords --data '{"keyword":"ラッコ","modes":["google"]}'
rakkokeyword raw GET /v1/metadata/locations --query countryCode=JP
```

## Output

```bash
rakkokeyword suggest-keywords ラッコ                 # table (humans; truncated, curated columns)
rakkokeyword suggest-keywords ラッコ -f json         # the API response, byte for byte
rakkokeyword suggest-keywords ラッコ -f jsonl        # one record per line
rakkokeyword suggest-keywords ラッコ -f csv          # every field, as dotted columns
rakkokeyword suggest-keywords ラッコ --fields keyword,metrics.searchVolume -f csv
```

Credits, progress and warnings go to stderr, so `> file` captures clean data.
`rakkokeyword auth set-format json` changes the default.

## Filtering

`--filter path=value`, repeatable; each command's `--help` lists the paths it
accepts, and unknown ones are rejected before a credit is spent.

```bash
rakkokeyword related-keywords ラッコ \
  --filter searchVolume.min=100 \
  --filter searchVolume.max=10000 \
  --filter keyword.notIncludes=グッズ,中古 \
  --sort-by searchVolume --order-by desc -n 1000
```

`--filter-json '{…}'` passes a raw filter object for anything `--filter` cannot
express.

## For AI agents

```bash
rakkokeyword llm                  # the full reference (rules, metrics, schemas, catalog)
rakkokeyword llm --format json    # the same, as chapters
```

The reference is embedded in the binary, so it works offline and always matches
the installed version. This repository also ships a Claude Code plugin
(`plugins/rakkokeyword-cli`) whose skills work in Copilot, Cursor and Gemini CLI
via `gh skill install`, and a `context7.json` for context7 MCP.

## Global flags

| Flag | Meaning |
| --- | --- |
| `--api-key` | override the resolved API key |
| `-f, --format` | `table` / `json` / `jsonl` / `csv` |
| `--fields` | choose columns as dotted paths |
| `--dry-run` | print the request and its cost, send nothing |
| `--wide` | do not truncate table cells |
| `--quiet` | suppress the credit notice |
| `--timeout` | HTTP timeout per request (default 2m) |
| `--retries` | extra attempts for 429 and 5xx (default 3) |

## Development

```bash
go generate ./...     # regenerate internal/llmdocs/90-commands.md
go test ./...         # includes plugin and SKILL.md validation
git diff --exit-code  # CI fails on a stale generated reference
```

## License

MIT © Ideamans Inc.
