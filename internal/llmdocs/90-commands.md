# Command catalog

Generated from the cobra command tree by `go generate ./...`.
Do not edit by hand — edit the command definitions instead.

## Global flags

| flag | type | default | description |
| --- | --- | --- | --- |
| `--api-key` | string | — | API key (overrides RAKKOKEYWORD_API_KEY and the config file) |
| `--dry-run` | bool | `false` | Print the request that would be sent, consume no credits, and exit |
| `--fields` | string | — | Comma-separated columns for table and csv, as dotted paths into a record (e.g. keyword,metrics.searchVolume) |
| `-f`, `--format` | string | — | Output format: table / json / jsonl / csv (default table, or the config file default) |
| `--quiet` | bool | `false` | Suppress the credit-consumption notice on stderr |
| `--retries` | int | `3` | Extra attempts for rate limits (429) and server errors (5xx) |
| `--timeout` | duration | `2m0s` | HTTP timeout per request |
| `--wide` | bool | `false` | Do not truncate long values in table output |

## `rakkokeyword auth`

Manage the rakkokeyword API key

The API key is resolved in this order: --api-key, then the
RAKKOKEYWORD_API_KEY environment variable, then RAKKO_API_KEY, then the
config file. Keys are issued on the STANDARD plan (up to 5) in the
rakkokeyword account settings.

### `rakkokeyword auth set-api-key`

Store the API key in the config file

Writes the key to the config file with owner-only permissions.

Prefer the RAKKOKEYWORD_API_KEY environment variable on shared machines
and in CI — it leaves nothing on disk.

```
rakkokeyword auth set-api-key <api-key>
```

### `rakkokeyword auth set-format`

Store the default output format (table / json / jsonl / csv)

```
rakkokeyword auth set-format <format>
```

### `rakkokeyword auth status`

Show where the API key comes from and which endpoint it talks to

## `rakkokeyword bulk-site-research`

Traffic, keyword and page scale of up to 100 sites at once (0.45 credits per URL)

Current scale and 12-month trend for many sites in one call: estimated
traffic, keyword count, page count, rank distribution and per-page averages.

The histories series are indices normalised to 100 at the series maximum —
etvIndex, keywordCountIndex and pageCountIndex — not absolute values. The
metrics block holds the real current numbers. Items come back in the same
order as the URLs given.

Requires the STANDARD plan or above. At most 100 URLs.

Cost: 0.45 credits per URL, minimum 4.5 credits (10 URLs → 4.5, 100 → 45).

```
rakkokeyword bulk-site-research [url...]
```

Aliases: bulk-sites

Example:

```
rakkokeyword bulk-site-research https://a.example/ https://b.example/
  rakkokeyword bulk-site-research --urls-file sites.txt --url-match-type sub_domain -f csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--url` | stringArray | `[]` | URL to research; repeat, or pass URLs as arguments |
| `--url-match-type` | string | — | Unit of research: url / forward_url / domain / sub_domain (API default: domain) |
| `--urls-file` | string | — | File with one URL per line (- for stdin) |

## `rakkokeyword co-occurrence`

Words that recur across the pages ranking for a keyword (3 credits)

The vocabulary of the top Google results for a keyword: how often each word
appears in body text, titles and headings, and on how many of the ranking
sites.

Words shared by most top pages are the ones an article on this topic is
expected to contain. siteCountTotal — how many ranking sites use the word at
all — is the most robust signal; a single verbose page can inflate the
occurrence counts on its own.

--details=false drops the per-page breakdown and makes the response much
smaller.

Cost: 3 credits per request.

```
rakkokeyword co-occurrence <keyword>
```

Aliases: cooc, cooccurrence

Example:

```
rakkokeyword co-occurrence ラッコ -n 30
  rakkokeyword cooc ラッコ --details=false -f csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--details` | bool | `true` | Include the per-page breakdown for every word (API default: true) |
| `-n`, `--limit` | int | `0` | Maximum records to return, any positive integer (API default: all results) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: word / occurrencePageCount / occurrenceTitleCount / occurrenceHeadingCount / siteCountTotal / siteCountHeading (API default: siteCountTotal) |

## `rakkokeyword competitive`

Sites whose ranking keywords overlap with a given site (4.5 credits)

Up to 20 sites that rank for the same keywords as the given domain, with
the overlap rate, estimated traffic, traffic value, keyword count and page
count of each.

duplicateRate is a fraction in [0,1] — 0.42 means 42% of keywords overlap.
competitorUniqueKeywordCount is the content gap: keywords they have and
the target does not.

Cost: 4.5 credits per request.

```
rakkokeyword competitive <url>
```

Aliases: competitors

Example:

```
rakkokeyword competitive https://example.com/
  rakkokeyword competitive https://example.com/ --sort-by duplicateRate -f json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: duplicate / duplicateRate / competitorUnique / targetUnique / etv / keywordCount / trafficValue / pageCount (API default: etv) |

## `rakkokeyword content-search`

Pages whose title, description or top keywords match a keyword (4.5 credits)

Finds web pages related to a keyword and reports their estimated traffic,
traffic value, ranking keyword count and best keyword. Up to 5,000 records.

Good for finding places to pitch a guest post or an ad, for competitor
content research, and — with --top-keyword-collapse — for surfacing niche
keywords that weak sites are ranking for unintentionally.

Cost: 4.5 credits per request.

```
rakkokeyword content-search <keyword>
```

Aliases: content

Example:

```
rakkokeyword content-search ラッコ --search-target title -n 100
  rakkokeyword content ラッコ --top-keyword-collapse --filter estimatedTraffic.min=500
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--advanced-search` | bool | `true` | Morphologically analyse the keyword for better matching (API default: true) |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: estimatedTraffic.min=<int> estimatedTraffic.max=<int> rankingKeywordCount.min=<int> rankingKeywordCount.max=<int> trafficValue.min=<int> trafficValue.max=<int> title.includes=<word,word> title.notIncludes=<word,word> url.includes=<word,word> url.notIncludes=<word,word> topKeyword.includes=<word,word> topKeyword.notIncludes=<word,word> description.includes=<word,word> description.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-5000 (API default: 100) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--search-target` | string | — | Where the keyword must appear: title / keyword / description / titleAndKeyword / titleAndKeywordAndDescription (API default: titleAndKeywordAndDescription) |
| `--sort-by` | string | — | Sort field: estimatedTraffic / trafficValue / rankingKeywordCount (API default: trafficValue) |
| `--top-keyword-collapse` | bool | `false` | Keep only one page per top keyword |

## `rakkokeyword headline`

Headings (h1-h6) of the pages ranking for a keyword (3 credits)

Extracts the headings of the top Google results for a keyword, plus each
page's character and heading count and the averages across them.

Topics that recur across the top pages are the ones Google's users are
assumed to need covered, which makes this the step before drafting a title
and outline. Pair it with `rakkokeyword co-occurrence` for the vocabulary.

Only h1-h4 are collected by default; add --h5 --h6 for the rest.

Cost: 3 credits per request.

```
rakkokeyword headline <keyword>
```

Aliases: headlines

Example:

```
rakkokeyword headline ラッコ
  rakkokeyword headline ラッコ --less-characters -f json | jq '.data.items[].headlines'
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--h1` | bool | `true` | Include <h1> headings (API default: true) |
| `--h2` | bool | `true` | Include <h2> headings (API default: true) |
| `--h3` | bool | `true` | Include <h3> headings (API default: true) |
| `--h4` | bool | `true` | Include <h4> headings (API default: true) |
| `--h5` | bool | `false` | Include <h5> headings (API default: false) |
| `--h6` | bool | `false` | Include <h6> headings (API default: false) |
| `--less-characters` | bool | `false` | Exclude pages with fewer than 1,000 characters |
| `--less-headlines` | bool | `false` | Exclude pages with fewer than 5 headings |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-20 (API default: 20) |
| `--order-by` | string | — | Sort order: asc / desc (API default: asc) |
| `--sort-by` | string | — | Sort field: position / title / headlineCount / wordCount (API default: position) |

## `rakkokeyword influx-keywords`

Keywords a site or page already earns Google traffic from (4.5 credits)

The keywords a domain or URL ranks for in Google, with its position and
estimated monthly traffic per keyword. Up to 10,000 records.

Run it on a competitor to see what they win on, on your own site to see
what you win on, and compare the two for the content gap.

Ranks and metrics may be stale; `rakkokeyword search-rank register` re-checks
positions and `rakkokeyword search-volume register` refreshes SEO metrics.

Cost: 4.5 credits per request.

Aliases: influx-kw

Example:

```
rakkokeyword influx-keywords --target https://example.com/ --match-type sub_domain -n 200
  rakkokeyword influx-keywords --target https://example.com/blog/post --match-type url --sort-by rank --order-by asc
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> rank.min=<int> rank.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> etv.min=<int> etv.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `--keyword-collapse` | bool | `false` | Collapse duplicate keywords across targets |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-10000 (API default: 100) |
| `--match-type` | string | — | How every --target matches: url / forward_url / domain / sub_domain (API default: sub_domain) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: keyword / seoDifficulty / rank / searchVolume / cpc / competition / etv (API default: etv) |
| `--target` | stringArray | `[]` | Domain or URL to investigate; repeat for up to 20 targets |
| `--targets-json` | string | — | Targets as raw JSON, for per-target match types: '[{"url":"https://a/","matchType":"url"}]' |

## `rakkokeyword influx-pages`

Pages of a site that earn the most Google traffic (4.5 credits)

The same data as `rakkokeyword influx-keywords` aggregated per page: total
estimated traffic, traffic value in USD, how many keywords the page ranks
for, and its single best keyword. Up to 10,000 records.

A competitor's top pages show which topics already have proven demand.
To see everything one of those pages ranks for, feed its URL back into
`rakkokeyword influx-keywords --match-type url`.

Cost: 4.5 credits per request.

Aliases: pages

Example:

```
rakkokeyword influx-pages --target https://example.com/ -n 50
  rakkokeyword influx-pages --target https://example.com/ --filter totalEtv.min=100 -f csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: totalEtv.min=<int> totalEtv.max=<int> keywordCount.min=<int> keywordCount.max=<int> totalTrafficValue.min=<int> totalTrafficValue.max=<int> title.includes=<word,word> title.notIncludes=<word,word> url.includes=<word,word> url.notIncludes=<word,word> topKeyword.includes=<word,word> topKeyword.notIncludes=<word,word> topSeoDifficulty.min=<int> topSeoDifficulty.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-10000 (API default: 100) |
| `--match-type` | string | — | How every --target matches: url / forward_url / domain / sub_domain (API default: sub_domain) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: totalEtv / totalTrafficValue / keywordCount (API default: totalEtv) |
| `--target` | stringArray | `[]` | Domain or URL to investigate; repeat for up to 20 targets |
| `--targets-json` | string | — | Targets as raw JSON, for per-target match types: '[{"url":"https://a/","matchType":"url"}]' |
| `--top-keyword-collapse` | bool | `false` | Collapse pages that share the same top keyword |

## `rakkokeyword metadata`

Region and language names accepted by --location and --language (free)

### `rakkokeyword metadata languages`

List language names for --language (free, no API key needed)

The language names `rakkokeyword search-volume register --language` and
`rakkokeyword search-rank register --language` accept. Use the value verbatim
(e.g. Japanese).

Cost: free. This endpoint needs no API key.

### `rakkokeyword metadata locations`

List region names for --location (free, no API key needed)

The region names `rakkokeyword search-volume register --location` and
`rakkokeyword search-rank register --location` accept.

Unfiltered the list is country-level only. Give --location-name or
--country-code and city-level regions appear too; those are written as
"City,Region,Country" (e.g. Shibuya,Tokyo,Japan). Intermediate levels on
their own — a prefecture with no city — are not supported.

Cost: free. This endpoint needs no API key.

Example:

```
rakkokeyword metadata locations --country-code JP
  rakkokeyword metadata locations --location-name Tokyo
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--country-code` | string | — | Filter by ISO 3166-1 alpha-2 country code (e.g. JP); also reveals city-level regions |
| `-n`, `--limit` | int | `0` | Maximum records to return (API default: all) |
| `--location-name` | string | — | Filter by region name (substring, case-insensitive); also reveals city-level regions |

## `rakkokeyword other-keywords`

LSI keywords and People-Also-Ask questions, recursively (22.5 credits)

What Google thinks someone searching this keyword will look for next
("People also search for", LSI) and what they are wondering about
("People also ask", PAA), gathered recursively up to two levels.

The importance field (high / medium / low) counts how often an entry
reappeared during the recursion — high means Google surfaces it broadly.

This is the most expensive per-request command in the CLI.

Cost: 22.5 credits per request.

```
rakkokeyword other-keywords <keyword>
```

Aliases: other, lsi, paa

Example:

```
rakkokeyword other-keywords ラッコ
  rakkokeyword other ラッコ -f json | jq '.data.items[] | select(.type=="paa")'
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: importance / seoDifficulty / searchVolume / cpc / competition / firstSeenRange (API default: importance) |

## `rakkokeyword question-search`

Frequently asked questions containing a keyword, by frequency (3 credits)

Questions from the rakkokeyword database that contain the keyword, ordered
by how often they occur. Up to 200 records.

Useful for FAQ and Q&A content, and for AIO / GEO / LLMO work: these are
the phrasings people are likely to type into an AI assistant.

For the questions Google itself shows on a SERP, use `rakkokeyword other-keywords`.

Cost: 3 credits per request.

```
rakkokeyword question-search <keyword>
```

Aliases: questions

Example:

```
rakkokeyword question-search ラッコ -n 50
  rakkokeyword questions ラッコ -f jsonl
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `-n`, `--limit` | int | `0` | Maximum questions to return, 1-200 (API default: 100) |

## `rakkokeyword ranking-keywords`

Keywords the pages ranking for this keyword also rank for (4.5 credits)

Takes the pages that rank highly for the keyword and reports the other
keywords those same pages rank for. Up to 5,000 records with SEO metrics.

relevance (1-100) is how much the two result sets overlap. High-relevance
keywords share search intent and can usually be targeted by one article;
low-relevance ones deserve their own.

Narrow --search-top and --search-range for closer intent, widen them to
discover keywords further afield.

Cost: 4.5 credits per request.

```
rakkokeyword ranking-keywords <keyword>
```

Aliases: ranking, co-ranking

Example:

```
rakkokeyword ranking-keywords ラッコ --search-top 10 --search-range 20
  rakkokeyword ranking ラッコ --filter relevance.min=50 -n 200
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> relevance.min=<int> relevance.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-5000 (API default: 500) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--search-range` | int | `0` | Rank cut-off for the keywords those pages rank for: 10 / 20 / 30 / 50 / 100 (API default: 50) |
| `--search-top` | int | `0` | How many top-ranking pages to inspect: 3 / 5 / 10 / 20 / 30 / 50 (API default: 20) |
| `--sort-by` | string | — | Sort field: seoDifficulty / searchVolume / cpc / competition / relevance (API default: relevance) |

## `rakkokeyword raw`

Call any API endpoint directly with a JSON body (cost: whatever that endpoint costs)

The escape hatch. Sends a request to any path of the rakkokeyword API and
prints the response, so a parameter this CLI has not wrapped is never a
dead end.

The body comes from --data, from @file, or from stdin when --data is "-".
Authentication, retries, timeouts and output formatting work as they do
everywhere else.

It charges the same credits as the endpoint it calls; --dry-run still shows
the request without sending it. The endpoint list is in `rakkokeyword llm` and at
https://api.rakkokeyword.com/api-docs.json.

```
rakkokeyword raw <METHOD> <path>
```

Example:

```
rakkokeyword raw POST /v1/suggest-keywords --data '{"keyword":"ラッコ","modes":["google"]}'
  rakkokeyword raw GET /v1/metadata/locations --query countryCode=JP
  rakkokeyword raw POST /v1/co-occurrence --data @body.json --items data.items -f csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `-d`, `--data` | string | — | JSON request body: inline, @file, or - for stdin |
| `--items` | string | `data.items` | Dotted path to the array to tabulate for table, csv and jsonl output |
| `--query` | stringArray | `[]` | Query parameter as key=value; repeatable |

## `rakkokeyword related-keywords`

Keywords from the rakkokeyword database matching a keyword, up to 25,000 (1.5 credits)

Bulk keyword harvesting: every keyword in the rakkokeyword database that
matches the given one, with SEO metrics, up to 25,000 records.

Reach for `rakkokeyword suggest-keywords` first — it reflects real search-engine
suggestions. Use this when you need volume beyond what suggestions give.

Cost: 1.5 credits per request.

```
rakkokeyword related-keywords <keyword>
```

Aliases: related

Example:

```
rakkokeyword related-keywords ラッコ --match-type phraseMatch -n 1000
  rakkokeyword related ラッコ --filter keyword.notIncludes=グッズ -f csv > keywords.csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> firstSeenRange.include=last_7_days\|last_30_days\|last_90_days\|within_6_months\|within_1_year\|over_1_year |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-25000 (API default: 1000) |
| `--match-type` | string | — | How the keyword must match: partialMatch / phraseMatch / prefixMatch / suffixMatch / wordMatch (API default: partialMatch) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: seoDifficulty / searchVolume / cpc / competition / firstSeenRange (API default: searchVolume) |

## `rakkokeyword search-rank`

Check live Google rankings of URLs for a list of keywords

Measures where URLs or domains currently rank in Google for given keywords,
as an asynchronous job: register, poll status, fetch results.

Unlike the ranks bundled with `rakkokeyword influx-keywords`, these are freshly
fetched SERPs for the region, language, device and OS you specify.

`rakkokeyword search-rank register --wait` runs all three steps in one go.

Aliases: rank

### `rakkokeyword search-rank histories`

List past rank checks, newest first (free)

Past rank checks with their requestId, status, keyword and URL summaries,
newest first. Use it to recover a requestId.

Cost: free.

Example:

```
rakkokeyword search-rank histories -n 20 --status completed
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-100 (API default: 100) |
| `--offset` | int | `0` | Records to skip (offset + limit must not exceed 50,000) |
| `--status` | string | — | Filter by status: completed / processing (API default: all) |

### `rakkokeyword search-rank register`

Register a rank check for keywords against URLs (0.9 credits/keyword for ranks 1-30)

Registers a rank check and prints the requestId. Every keyword is checked
against every URL, so 10 keywords and 3 URLs is one job covering 30 pairs —
but the cost is per keyword, not per pair.

Without --wait, follow up with `rakkokeyword search-rank status <requestId>` and
then `rakkokeyword search-rank results <requestId>`. With --wait this command
polls and prints the results itself.

Timing: up to 10 keywords usually finishes in minutes, larger jobs within
about an hour, and a busy queue can take longer.

Cost: 0.9 credits per keyword for ranks 1-30, plus 0.3 per keyword for each
additional 10 ranks of --depth. --search-volume adds the metrics of
`rakkokeyword search-volume` on top.

```
rakkokeyword search-rank register [keyword...]
```

Example:

```
rakkokeyword search-rank register ラッコ カワウソ --url https://example.com/ --wait
  rakkokeyword search-rank register --keywords-file kw.txt --url https://example.com/ --depth 100 --device mobile
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--deduplicate` | bool | `true` | Drop duplicate keywords before charging for them (API default: true) |
| `--depth` | int | `0` | How deep to read the SERP: 30 / 40 / … / 100 (API default: 30) |
| `--device` | string | — | Device to emulate: desktop / mobile (API default: desktop) |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `--keyword` | stringArray | `[]` | Keyword to check; repeat, or pass keywords as arguments |
| `--keywords-file` | string | — | File with one keyword per line (- for stdin) |
| `--language` | string | — | Language name for the SERP, from `rakkokeyword metadata languages` (API default: Japanese) |
| `-n`, `--limit` | int | `0` | Maximum records to return, any positive integer (API default: 100) |
| `--location` | string | — | Region name for the SERP, from `rakkokeyword metadata locations` (API default: Japan) |
| `--match-type` | string | — | How a result counts as a hit: url / forward_url / domain / sub_domain (API default: sub_domain) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--os` | string | — | OS to emulate: windows / macos (desktop) or android / ios (mobile) |
| `--poll-interval` | duration | `30s` | How often to poll while waiting (the API recommends 30s) |
| `--search-volume` | bool | `false` | Also fetch search volume and SEO difficulty for every keyword (extra credits) |
| `--sort-by` | string | — | Sort field: keyword / seoDifficulty / searchVolume (API default: searchVolume) |
| `--url` | stringArray | `[]` | URL or domain to look for in the results; repeat, up to 50 |
| `--urls-file` | string | — | File with one URL per line |
| `--wait` | bool | `false` | Poll until the job completes; the result flags below then shape the fetched results |
| `--wait-timeout` | duration | `1h0m0s` | Give up waiting after this long; the job keeps running and can be fetched later by requestId |
| `--with-aggregation` | bool | `false` | Include per-target totals (estimated traffic, rank distribution) in the summary |

### `rakkokeyword search-rank results`

Fetch the results of a completed rank check (free)

Fetches the ranks for a requestId from `rakkokeyword search-rank register`.

Each item carries a rankings array with one entry per target URL. A null
position means the URL was not found within --depth — that is "not in the
top N", not "rank 0". In table output the rankings array is shown as JSON;
use -f json or --fields to work with it properly.

Cost: free.

```
rakkokeyword search-rank results <requestId>
```

Example:

```
rakkokeyword search-rank results 01HQZX5Y4JMQK8XNQ7WVZXZ5Y4
  rakkokeyword search-rank results 01HQZX… --with-aggregation -f json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, any positive integer (API default: 100) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: keyword / seoDifficulty / searchVolume (API default: searchVolume) |
| `--with-aggregation` | bool | `false` | Include per-target totals (estimated traffic, rank distribution) in the summary |

### `rakkokeyword search-rank status`

Check whether a rank check has finished (free)

Reports isCompleted plus the per-stage statuses. Poll every 30 seconds;
--wait does that for you.

A searchVolumeAndSeoDifficulty status of failed or integration_failed keeps
isCompleted false — the SERP ranks may still be fetchable.

Cost: free.

```
rakkokeyword search-rank status <requestId>
```

Example:

```
rakkokeyword search-rank status 01HQZX5Y4JMQK8XNQ7WVZXZ5Y4 --wait
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--poll-interval` | duration | `30s` | How often to poll while waiting (the API recommends 30s) |
| `--wait` | bool | `false` | Poll until the job completes |
| `--wait-timeout` | duration | `1h0m0s` | Give up waiting after this long; the job keeps running and can be fetched later by requestId |

## `rakkokeyword search-volume`

Batch keyword metrics: fresh search volume, SEO difficulty, CPC, trends

Batch investigation of a keyword list — monthly search volume, SEO
difficulty, CPC, competition and month-by-month trends — as an asynchronous
job: register, poll status, fetch results.

This is the only command that returns freshly measured metrics. Everywhere
else the metrics ride along with whatever was last cached.

`rakkokeyword search-volume register --wait` runs all three steps in one go.

Aliases: volume

### `rakkokeyword search-volume histories`

List past batch keyword investigations, newest first (free)

Past requests with their requestId, status and keyword summary, newest
first. Use it to recover a requestId whose results were never fetched.

Cost: free.

Example:

```
rakkokeyword search-volume histories -n 20
  rakkokeyword volume histories --status processing
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-100 (API default: 100) |
| `--offset` | int | `0` | Records to skip (offset + limit must not exceed 50,000) |
| `--status` | string | — | Filter by status: completed / processing (API default: all) |

### `rakkokeyword search-volume register`

Register a batch keyword investigation (0.03 credits/keyword, 0.78 with --seo-difficulty, minimum 15)

Registers up to 50,000 keywords for investigation and prints the requestId.

Without --wait, follow up with `rakkokeyword search-volume status <requestId>` and
then `rakkokeyword search-volume results <requestId>`. With --wait this command
polls the status itself and prints the results when they are ready; the
result-shaping flags (--limit, --sort-by, --filter, --noise-reduction)
apply to that final fetch.

Timing: usually about 10 seconds, but --seo-difficulty pushes it to as much
as 60 minutes, and a busy queue can take hours. Leave --seo-difficulty off
unless the difficulty score is what you came for.

Cost: 0.03 credits per keyword, plus 0.75 per keyword with --seo-difficulty.
A request always costs at least 15 credits, so batching pays.

```
rakkokeyword search-volume register [keyword...]
```

Example:

```
rakkokeyword search-volume register ラッコ カワウソ --wait
  rakkokeyword search-volume register --keywords-file keywords.txt --seo-difficulty
  rakkokeyword volume register --keywords-file - --location Japan --language Japanese
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--aggregation-period-months` | int | `0` | Trend window in months: 12 / 24 / 36 / 48 (API default: 12) |
| `--data-completion` | bool | `true` | Fill gaps in the volume data (API default: true) |
| `--deduplicate` | bool | `true` | Drop duplicate keywords before charging for them (API default: true) |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> firstSeenRange.include=last_7_days\|last_30_days\|last_90_days\|within_6_months\|within_1_year\|over_1_year |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `--keyword` | stringArray | `[]` | Keyword to investigate; repeat, or pass keywords as arguments |
| `--keywords-file` | string | — | File with one keyword per line (- for stdin); up to 50,000 |
| `--language` | string | — | Language name, from `rakkokeyword metadata languages` (API default: Japanese) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-50000 (API default: 100) |
| `--location` | string | — | Region name, from `rakkokeyword metadata locations` (API default: Japan) |
| `--noise-reduction` | bool | `true` | Apply noise reduction to the result set (API default: true) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--poll-interval` | duration | `30s` | How often to poll while waiting (the API recommends 30s) |
| `--seo-difficulty` | bool | `false` | Also measure SEO difficulty — 25x the cost and up to 60 minutes slower |
| `--sort-by` | string | — | Sort field: keyword / seoDifficulty / searchVolume / rateOfChange / cpc / competition / firstSeenRange (API default: searchVolume) |
| `--wait` | bool | `false` | Poll until the job completes; the result flags below then shape the fetched results |
| `--wait-timeout` | duration | `1h0m0s` | Give up waiting after this long; the job keeps running and can be fetched later by requestId |

### `rakkokeyword search-volume results`

Fetch the results of a completed batch keyword investigation (free)

Fetches the data for a requestId from `rakkokeyword search-volume register`.
Check `rakkokeyword search-volume status <requestId>` first — results before
completion are partial.

Cost: free.

```
rakkokeyword search-volume results <requestId>
```

Example:

```
rakkokeyword search-volume results 1234567 -n 500 -f csv
  rakkokeyword volume results 1234567 --filter searchVolume.min=100 --sort-by searchVolume
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> firstSeenRange.include=last_7_days\|last_30_days\|last_90_days\|within_6_months\|within_1_year\|over_1_year |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-50000 (API default: 100) |
| `--noise-reduction` | bool | `true` | Apply noise reduction to the result set (API default: true) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: keyword / seoDifficulty / searchVolume / rateOfChange / cpc / competition / firstSeenRange (API default: searchVolume) |

### `rakkokeyword search-volume status`

Check whether a batch keyword investigation has finished (free)

Reports isCompleted plus the per-stage statuses. Poll every 30 seconds;
--wait does that for you and returns when the job is done.

noiseReduction is not part of the completion test — isCompleted can be true
while it is still processing.

Cost: free.

```
rakkokeyword search-volume status <requestId>
```

Example:

```
rakkokeyword search-volume status 1234567
  rakkokeyword search-volume status 1234567 --wait
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--poll-interval` | duration | `30s` | How often to poll while waiting (the API recommends 30s) |
| `--wait` | bool | `false` | Poll until the job completes |
| `--wait-timeout` | duration | `1h0m0s` | Give up waiting after this long; the job keeps running and can be fetched later by requestId |

## `rakkokeyword site-search`

Find sites by content, domain or SEO scale (1.5 credits)

Searches whole sites rather than pages, ordered by estimated traffic, and
reports traffic, traffic value, ranking keyword count and page count.
At most 100 records.

With a content filter (filter.keyword.includes) the API first takes the
100 most-trafficked related sites and only then applies the other filters,
so filtering cannot page past the first 100 — narrow the content filter
instead. A content filter also adds relatedContent metrics to each record.

Cost: 1.5 credits per request.

Aliases: sites

Example:

```
rakkokeyword site-search --filter keyword.includes=ラッコ -n 20
  rakkokeyword site-search --filter domain.includes=.jp --filter totalEtv.min=10000 -f json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: keyword.includes=<word,word> keyword.notIncludes=<word,word> domain.includes=<word,word> domain.notIncludes=<word,word> domain.matchType=partialMatch\|prefixMatch\|suffixMatch totalEtv.min=<int> totalEtv.max=<int> keywordCount.min=<int> keywordCount.max=<int> pageCount.min=<int> pageCount.max=<int> totalTrafficValue.min=<int> totalTrafficValue.max=<int> relatedContentEtv.min=<int> relatedContentEtv.max=<int> contentRelevance.min=<int> contentRelevance.max=<int> |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `-n`, `--limit` | int | `0` | Maximum records to return, 1-100 (API default: 100) |

## `rakkokeyword suggest-keywords`

Search-engine suggestions for a keyword, with SEO metrics (1.5 credits)

Autocomplete suggestions for a keyword from Google, Bing, YouTube, Amazon,
Rakuten and others, with monthly search volume, SEO difficulty, CPC and
competition attached.

This is the usual first step of keyword research: it shows the compound
queries real users type. About 1,000 suggestions are available normally and
about 10,000 with --increase-keyword.

The attached SEO metrics may be stale. When they matter, feed the keywords
into `rakkokeyword search-volume register` for fresh figures.

Cost: 1.5 credits per request.

```
rakkokeyword suggest-keywords <keyword>
```

Aliases: suggest

Example:

```
rakkokeyword suggest-keywords ラッコ --modes google,bing -n 50
  rakkokeyword suggest ラッコ --increase-keyword --filter searchVolume.min=100 -f json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--filter` | stringArray | `[]` | Filter as path=value, repeatable. Accepted paths: suggestClass=<n,n> keyword.includes=<word,word> keyword.notIncludes=<word,word> seoDifficulty.min=<int> seoDifficulty.max=<int> searchVolume.min=<int> searchVolume.max=<int> cpc.min=<number> cpc.max=<number> competition.min=<int> competition.max=<int> firstSeenRange.include=last_7_days\|last_30_days\|last_90_days\|within_6_months\|within_1_year\|over_1_year |
| `--filter-json` | string | — | Filter object as raw JSON, merged over --filter (escape hatch for anything --filter cannot express) |
| `--increase-keyword` | bool | `false` | Fetch the extended suggestion set (~10,000 instead of ~1,000) |
| `-n`, `--limit` | int | `0` | Maximum records to return, any positive integer (API default: all results) |
| `--modes` | stringSlice | `[]` | Search engines to pull suggestions from, comma-separated: google / bing / youtube / googleVideo / amazon / rakuten / googleShopping / googleImage (API default: google) |
| `--order-by` | string | — | Sort order: asc / desc (API default: desc) |
| `--sort-by` | string | — | Sort field: keyword / suggestClass / seoDifficulty / searchVolume / cpc / competition / firstSeenRange (API default: searchVolume) |
