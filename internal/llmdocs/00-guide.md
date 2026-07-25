# rakko — reference for AI agents

`rakko` is a CLI for the ラッコキーワード (rakkokeyword) API: keyword research,
SEO metrics, SERP rankings, competitor and content analysis, mostly for the
Japanese search market.

## How to read this reference

**Do not read all of it.** The whole thing is about 1,450 lines, most of which
is the command catalog. This first chapter is the only one worth reading end to
end; the rest are lookups.

| Chapter | Read it when |
| --- | --- |
| `00-guide.md` (this one) | always — rules, auth, which command answers what |
| `10-metrics.md` | before quoting a number: units, ranges, how fresh it is |
| `20-schemas.md` | before parsing: the JSON shape of each response |
| `30-gotchas.md` | before a big or repeated call: costs, limits, async behaviour |
| `90-commands.md` | rarely — `rakko <command> --help` says the same thing for one command |

Pull one chapter instead of the whole reference:

```bash
rakko llm --format json | jq -r '.[] | select(.file=="20-schemas.md") | .body'
rakko llm | sed -n '/^# Limits, costs and traps/,$p'
rakko suggest-keywords --help          # flags, defaults and cost for one command
```

## The four rules

1. **Every call costs the user money.** Credits come out of the account the API
   key belongs to. Costs range from free (metadata, histories, status, results)
   through 1.5 credits (suggest, related, site-search) up to 22.5 credits for a
   single `other-keywords` call. Say what a command will cost before running it,
   and use `--dry-run` to check a request without spending anything.
2. **Always pass `-f json` when parsing.** The default `table` format is for
   humans: it truncates long values and shows a subset of columns. `-f json`
   prints the API response byte-for-byte; `-f jsonl` prints one record per line;
   `-f csv` flattens every field into dotted columns.
3. **`null` means "unknown", not zero.** `seoDifficulty` is often null, and a
   null `position` in rank results means "not found within --depth". Never
   report either as 0.
4. **The batch commands are asynchronous.** `search-volume` and `search-rank`
   are register → status → results. Use `register --wait` unless you have a
   reason not to.

## Authentication

The API key is resolved in this order:

1. `--api-key`
2. `RAKKOKEYWORD_API_KEY` (preferred; nothing is written to disk)
3. `RAKKO_API_KEY`
4. the config file (`rakko auth set-api-key <key>`)

`rakko auth status` reports which source is in play without revealing the key.
Keys are issued from a rakkokeyword STANDARD plan or above (up to 5 per
account). `rakko metadata locations` and `rakko metadata languages` are the only
commands that work without a key.

## Which command answers which question

| The user wants | Command | Cost |
| --- | --- | --- |
| Broad related keywords, real search suggestions | `suggest-keywords` | 1.5 |
| Tens of thousands of keywords containing a word | `related-keywords` | 1.5 |
| What people search or ask *next* (LSI / PAA) | `other-keywords` | 22.5 |
| Question phrasings for FAQ / AI-search content | `question-search` | 3 |
| Keywords with the same search intent | `ranking-keywords` | 4.5 |
| Accurate, current volume / difficulty for a list | `search-volume register --wait` | 0.03–0.78 per keyword, min 15 |
| Where a site ranks right now | `search-rank register --wait` | 0.9+ per keyword |
| What keywords a site (or competitor) wins on | `influx-keywords` | 4.5 |
| Which pages of a site bring the traffic | `influx-pages` | 4.5 |
| Who the SEO competitors are | `competitive` | 4.5 |
| Scale and trend of many sites at once | `bulk-site-research` | 0.45 per URL, min 4.5 |
| Pages covering a topic (guest posts, research) | `content-search` | 4.5 |
| Sites covering a topic | `site-search` | 1.5 |
| What headings a ranking article needs | `headline` | 3 |
| What vocabulary a ranking article needs | `co-occurrence` | 3 |
| Valid `--location` / `--language` values | `metadata locations` / `metadata languages` | free |
| Anything not wrapped above | `raw <METHOD> <path>` | that endpoint's cost |

## Output contract

Every response shares one envelope:

```json
{ "result": true, "meta": { "consumedCredit": 1.5 }, "data": { }, "errors": [] }
```

- `-f json` prints that whole envelope, unmodified.
- `-f jsonl` and `-f csv` print the records under `data.items` (or
  `data.locations` / `data.languages` for metadata).
- Credit consumption and progress messages go to **stderr**, so stdout stays
  parseable.
- A non-2xx response is an error with a non-zero exit status; the API's own
  message is included.

## Choosing columns

Table and CSV columns are dotted paths into a record — `metrics.searchVolume`,
`ranking.position`, `page.url`. `--fields` overrides them:

```bash
rakko suggest-keywords ラッコ --fields keyword,metrics.searchVolume -f csv
```

Table output shows a curated subset; CSV shows every field the response has.

## Filtering

`--filter path=value`, repeatable, with the accepted paths listed in each
command's `--help`. Repeating a list-valued path appends to it.

```bash
rakko related-keywords ラッコ \
  --filter searchVolume.min=100 --filter searchVolume.max=10000 \
  --filter keyword.notIncludes=グッズ,中古
```

Unknown paths are rejected locally, before a credit is spent. `--filter-json`
takes a raw JSON filter object for anything `--filter` cannot express.

## Worked example: brief an SEO article

```bash
# 1. Map the demand (1.5 credits)
rakko suggest-keywords ラッコ --increase-keyword --filter searchVolume.min=100 -f json > suggest.json

# 2. Confirm the numbers on the shortlist (min 15 credits, ~10 seconds)
rakko search-volume register --keywords-file shortlist.txt --wait -f json > volume.json

# 3. See what the ranking pages cover (3 + 3 credits)
rakko headline ラッコ -f json > headlines.json
rakko co-occurrence ラッコ --details=false -f json > vocabulary.json

# 4. Answer the questions readers actually have (3 credits)
rakko question-search ラッコ -n 50 -f json > questions.json
```

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| `no API key` | nothing configured | `export RAKKOKEYWORD_API_KEY=…` or `rakko auth set-api-key` |
| HTTP 403 | wrong or revoked key | check `rakko auth status`, reissue the key |
| HTTP 402 | account out of credits | nothing the CLI can do; the user must top up |
| HTTP 429 | rate limit | already retried with backoff; space calls out or lower concurrency |
| HTTP 400 | a parameter the API rejected | re-read the command's `--help`; the message names the field |
| `unknown filter path` | filter not valid for this endpoint | the error lists the accepted paths |
| status never completes | busy queue | jobs can take hours; the requestId stays valid — fetch it later |
| empty `data.items` | filters too tight, or no data | loosen the filter before assuming the site or keyword has nothing |
