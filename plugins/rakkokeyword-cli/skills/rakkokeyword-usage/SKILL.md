---
name: rakkokeyword-usage
description: Research keywords and SEO with the rakkokeyword CLI for the rakkokeyword (ラッコキーワード) API — search-engine suggestions, monthly search volume and SEO difficulty, Google rank checks, competitor and content-gap analysis, and the headings and vocabulary of pages ranking for a keyword. Use when the user asks what people search for, how much volume or difficulty a keyword has, where a site ranks, who its SEO competitors are, or what an article about a topic should cover — especially for the Japanese market.
license: MIT
compatibility: Requires the `rakkokeyword` binary on PATH — run the rakkokeyword-install skill if it is missing — and a rakkokeyword API key (STANDARD plan or above) in RAKKOKEYWORD_API_KEY. Every call except metadata, histories, status and results spends the user's credits.
allowed-tools: Bash(rakkokeyword:*) Bash(jq:*) Bash(command:*) Read Write
---

# rakkokeyword-usage

Keyword and SEO research through the rakkokeyword API.

## 1. Confirm the tool and the key

```bash
command -v rakkokeyword && rakkokeyword auth status
```

Missing binary? Run the `rakkokeyword-install` skill. No key? Tell the user to set
`RAKKOKEYWORD_API_KEY`, issued in the rakkokeyword account settings (STANDARD
plan or above). Never echo a key back into the conversation.

## 2. Spend the user's credits deliberately

Everything except `metadata`, `histories`, `status` and `results` bills the
user. `rakkokeyword <command> --help` states the exact cost and `--dry-run` prints the
request with its price without sending it. Say the cost before running, and get
agreement above a few credits.

Three cases cause most of the waste:

- `other-keywords` costs **22.5 credits per call** — never loop it over a list.
- `search-volume register` has a **15-credit minimum per request** — collect the
  keywords and register one batch.
- `--seo-difficulty` multiplies the per-keyword price by 25 and can add an hour.

## 3. Pick the command by the question

| The user asks | Command |
| --- | --- |
| "what do people search around X?" | `rakkokeyword suggest-keywords X` |
| "give me every keyword containing X" | `rakkokeyword related-keywords X -n 5000` |
| "what do they search or ask next?" | `rakkokeyword other-keywords X` (expensive) |
| "which keywords can one article cover?" | `rakkokeyword ranking-keywords X` |
| "how much volume / difficulty really?" | `rakkokeyword search-volume register --keywords-file kw.txt --wait` |
| "where do we rank for these?" | `rakkokeyword search-rank register --keywords-file kw.txt --url https://site/ --wait` |
| "what does competitor X rank for?" | `rakkokeyword influx-keywords --target https://x/` |
| "who are our SEO competitors?" | `rakkokeyword competitive https://site/` |
| "what should this article contain?" | `rakkokeyword headline X` + `rakkokeyword co-occurrence X` |

`rakkokeyword --help` lists the rest (questions, pages, sites, bulk site research), and
`rakkokeyword raw POST /v1/… --data '{…}'` reaches anything unwrapped.

## 4. Read only the reference you need

`rakkokeyword llm` is ~1,450 lines — do not read it whole. Take the first chapter for
orientation and pull the others on demand:

```bash
rakkokeyword llm | sed -n '1,/^# 指標/p'                                                # rules, auth, command map
rakkokeyword llm --format json | jq -r '.[] | select(.file=="20-schemas.md") | .body'   # response shapes
rakkokeyword suggest-keywords --help                                                    # one command's flags and cost
```

## 5. Query with `-f json`, then interpret carefully

```bash
rakkokeyword suggest-keywords ラッコ --filter searchVolume.min=100 -n 200 -f json
```

Parse only `json` / `jsonl` / `csv` — the default table truncates cells and
shows a curated subset of columns. Credits and progress go to stderr.

The mistakes that produce confidently wrong SEO advice:

- **Only `search-volume` and `search-rank` measure anything.** Metrics from
  every other command are cached, often stale, and their `seoDifficulty` is
  usually `null`. Discover cheaply, then re-measure the shortlist.
- **`null` is not 0.** A null rank `position` means "not in the top `--depth`".
- **`cpc` and `trafficValue` are USD**, and volumes are Google Ads buckets (90,
  480, 1600, 18100 …) — equal values mean the same band, not equal demand.
- **Compare `data.summary.totalCount` with `returnedCount`** before saying a
  site "only ranks for N keywords"; the limit may have truncated it.

`10-metrics.md` has the rest (index-valued history series, fraction-valued
rates). When reporting, give the keyword, the market (`--location` /
`--language`, default Japan / Japanese) and whether the number was measured or
cached — a volume without its market is not actionable.

## 6. Batch jobs are asynchronous

`register --wait` does register → poll → results in one command. If waiting
times out the job keeps running: note the `requestId` and fetch it later with
`rakkokeyword search-volume results <id>`, or find it with `histories`.

## Failure modes

| Symptom | Fix |
| --- | --- |
| `command not found: rakkokeyword` | run the `rakkokeyword-install` skill |
| `no API key` / HTTP 403 | `export RAKKOKEYWORD_API_KEY=…`, or reissue the key |
| HTTP 402 | out of credits; the user must top up |
| HTTP 429 | already retried with backoff — space the calls out |
| `unknown filter path` | the error lists the paths this command accepts |
| status never completes | busy queue; the requestId stays valid, fetch it later |
| empty `items` | loosen the filter before concluding there is no data |
