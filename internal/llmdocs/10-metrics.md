# Metrics and how to read them

Every keyword-shaped response carries the same `metrics` block. Getting the
units and the freshness right is most of the work of using this data well.

| Field | Meaning | Range | Notes |
| --- | --- | --- | --- |
| `searchVolume` | monthly searches, averaged over the aggregation window | 0+ | `null` when unknown. Google Ads rounds these into buckets (10, 90, 480, 1600, 18100, 90500 …); treat them as orders of magnitude, not measurements |
| `seoDifficulty` | how hard it is to rank | 1–100 | 1–33 low, 34–66 medium, 67–100 high. **Frequently `null`** — it is only measured on request in `search-volume --seo-difficulty` |
| `cpc` | estimated cost per click | 0+ | **US dollars**, not yen |
| `competition` | paid-search competition | 0–100 | 0–33 low, 34–66 medium, 67–100 high. This is an *advertising* signal, not an SEO one |
| `firstSeenRange` | when the keyword first appeared in the rakkokeyword database | label | `last_7_days` … `over_1_year`. A recent value on a high-volume keyword suggests a trend |
| `relevance` | intent overlap with the source keyword (`ranking-keywords`) | 1–100 | High ⇒ one article can target both |
| `importance` | how often an LSI/PAA entry recurred (`other-keywords`) | high/medium/low | Derived from the recursive crawl, not from search volume |

## Freshness: which commands measure, which recall

Only `search-volume` measures. Every other command attaches whatever metrics
were last cached, and the API documentation says so explicitly for each one.

```
suggest-keywords / related-keywords / other-keywords / ranking-keywords
influx-keywords / influx-pages / content-search / site-search
        └── metrics may be stale, seoDifficulty usually null
                        │
                        ▼
        search-volume register --wait   ← current volume, CPC, competition,
                                          trends, and (opt-in) SEO difficulty
        search-rank register --wait     ← current Google positions
```

So the honest workflow is: discover cheaply and broadly, then re-measure the
shortlist. Quoting a difficulty score straight out of `suggest-keywords` is the
most common way to be confidently wrong with this API.

## Traffic metrics

| Field | Meaning |
| --- | --- |
| `estimatedTraffic` / `etv` | estimated monthly search visits |
| `trafficValue` | estimated traffic × CPC, i.e. what the traffic would cost as ads, **in USD** |
| `rankingKeywordCount` / `keywordCount` | how many keywords the page or site ranks for |
| `pageCount` | indexed pages known to rakkokeyword |
| `duplicateRate` (`competitive`) | keyword overlap as a fraction in [0,1] — 0.42 is 42% |
| `pagesWithTrafficRate` (`bulk-site-research`) | fraction of pages that get any traffic, in [0,1] |

These are estimates derived from rank × click-through model, not analytics.
Compare sites against each other rather than treating a number as a forecast.

## bulk-site-research indices

`histories[]` is normalised: `etvIndex`, `keywordCountIndex` and
`pageCountIndex` are 0–100 with the series maximum at 100, computed
independently per series and per site. They show shape, not size.

Absolute current values live in `metrics` (`estimatedTraffic`, `keywordCount`,
`pageCount`). Never compare an index across sites as if it were a volume, and
never multiply an index by anything.

## Rank data

`search-rank` results carry a `rankings` array with one entry per target URL:

- `position` — `null` when the URL did not appear within `--depth`. That is
  "not in the top N", which is different from "rank 0" and different from
  "no data".
- `rankedUrl` — the URL that actually ranked, which may not be the one asked
  about (a different page of the same domain, with `sub_domain` matching).
- `estimatedTraffic` — modelled from position and volume.

`influx-keywords` positions come from the crawl cache instead, so they can lag.
When the user asks "where do we rank *now*", use `search-rank`.

## suggestClass (suggest-keywords)

| Label | Meaning |
| --- | --- |
| `＋` | a direct suggestion |
| `＋＋` | a suggestion of a suggestion |
| `＋α` | suggestions found by appending あいうえお / abc / 123 to the keyword |
| `＋＋＋` | expanded further from `＋＋` or `＋α` |

`＋` entries are the closest to what users actually type; the rest come from
`--increase-keyword`-style expansion and get progressively more speculative.
