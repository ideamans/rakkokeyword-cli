# Limits, costs and traps

## Credits

| Command | Cost |
| --- | --- |
| `suggest-keywords`, `related-keywords`, `site-search` | 1.5 per request |
| `question-search`, `headline`, `co-occurrence` | 3 per request |
| `ranking-keywords`, `influx-keywords`, `influx-pages`, `competitive`, `content-search` | 4.5 per request |
| `other-keywords` | **22.5 per request** |
| `bulk-site-research` | 0.45 per URL, minimum 4.5 |
| `search-volume register` | 0.03 per keyword, **+0.75 per keyword** with `--seo-difficulty`, minimum 15 per request |
| `search-rank register` | 0.9 per keyword for ranks 1–30, +0.3 per keyword per extra 10 ranks of `--depth` |
| `metadata`, `histories`, `status`, `results` | free |

Consequences worth acting on:

- **Batch the batch commands.** A `search-volume register` with one keyword and
  one with 500 both cost at least 15 credits. Collect the shortlist first.
- **`--seo-difficulty` is 25× the price** and can take an hour. Leave it off
  unless difficulty is the question.
- **`other-keywords` costs as much as fifteen `suggest-keywords` calls.** Do not
  loop it over a keyword list.
- **`--dry-run` costs nothing** and prints the exact request plus its price.
- Consumed credits are printed to stderr after every call, and are in
  `meta.consumedCredit` in `-f json`.

## Hard limits

| Limit | Where |
| --- | --- |
| 20 targets | `influx-keywords`, `influx-pages` |
| 50 URLs | `search-rank register --url` |
| 100 URLs | `bulk-site-research` (STANDARD plan or above) |
| 100 records | `site-search`, and `histories` per page |
| 200 records | `question-search` |
| 5,000 records | `ranking-keywords`, `content-search` |
| 10,000 records | `influx-keywords`, `influx-pages` |
| 25,000 records | `related-keywords` |
| 50,000 keywords | `search-volume register` |
| offset + limit ≤ 50,000 | `histories` |

The CLI rejects the target/URL overruns locally; the rest come back as HTTP 400.

## Asynchronous jobs

```
register  →  requestId  →  status (poll)  →  results
```

- Poll no faster than every 30 seconds. `--wait` defaults to that.
- Typical `search-volume` completion is ~10 seconds; with `--seo-difficulty` up
  to 60 minutes. `search-rank` is minutes for ≤10 keywords, up to an hour
  beyond. A busy queue can mean hours for either.
- `--wait-timeout` (default 1 hour) only stops the CLI waiting. The job keeps
  running: recover it with `rakko search-volume histories` and fetch the
  results by requestId whenever it finishes.
- Fetching results before completion returns partial data without an error.
  Check `isCompleted` first.
- `isCompleted` ignores `noiseReduction` for search-volume; for search-rank a
  `failed` or `integration_failed` metrics stage keeps it false even though the
  SERP ranks may be there.

## Data traps

- **`null` ≠ 0.** Null `seoDifficulty` means not measured. Null `position` means
  not found within `--depth`.
- **`cpc` and `trafficValue` are USD**, everything else about the Japanese
  market notwithstanding.
- **Search volumes are Google Ads buckets** (90, 480, 1600, 18100, 90500 …).
  Two keywords sharing a volume are in the same band, not equally popular.
- **`duplicateRate` and `pagesWithTrafficRate` are fractions in [0,1].**
  Multiply by 100 before writing a percentage.
- **`bulk-site-research` histories are 0–100 indices**, not traffic.
- **`totalCount` vs `returnedCount`.** A limit silently truncates; compare the
  two before saying "the site ranks for 100 keywords".
- **`site-search` with a content filter cannot page past 100.** The API picks
  the top 100 sites *first* and filters afterwards.
- **`ranking-keywords --search-top/--search-range` change what the result
  means.** Wide ranges surface loosely related keywords by design.
- **`search-rank` matching.** `sub_domain` (the default) counts any subdomain;
  check `rankedUrl` to see which page actually ranked.

## Output traps

- Table output truncates cells to 44 characters (`--wide` disables it) and shows
  a curated column subset. Never scrape it — use `-f json`, `-f jsonl` or
  `-f csv`.
- CSV columns are dotted paths in the API's own field order; nested arrays of
  objects (`headlines`, `rankings`, `pageDetails`, `histories`) become one
  column of compact JSON.
- Credits, progress and API warnings go to stderr. Redirecting only stdout keeps
  the data clean.
- `--fields` accepts any dotted path from the schemas chapter and applies to
  both table and CSV.

## Regions and languages

`--location` and `--language` take names, not codes: `Japan`, `Japanese`. City
level works as `City,Region,Country` (`Shibuya,Tokyo,Japan`); a prefecture on
its own does not. Confirm with `rakko metadata locations --country-code JP`,
which is free and needs no API key.

## When a parameter is missing from the CLI

`rakko raw <METHOD> <path> --data '{…}'` sends anything to any endpoint with the
same auth, retries and formatting. The API's own spec is at
<https://api.rakkokeyword.com/api-docs.json>.
