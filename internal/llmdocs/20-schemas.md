# JSON output schemas

`-f json` prints the API response unchanged, so these are the API's own shapes.
`-f jsonl` prints the elements of the array marked **items path** below, one per
line. `-f csv` prints those same elements with dotted column names.

## The envelope

```json
{
  "result": true,
  "meta": { "consumedCredit": 1.5 },
  "data": { "query": {}, "summary": {}, "items": [] },
  "errors": []
}
```

`data.summary.totalCount` is how many records matched; `returnedCount` is how
many came back under the limit. When they differ, the result is truncated —
raise `-n` or narrow the filter rather than reporting the subset as the whole.

## suggest-keywords — items path `data.items`

```json
{
  "keyword": "ラッコ 水族館",
  "suggestClass": "＋",
  "metrics": { "seoDifficulty": 36, "searchVolume": 18100, "cpc": 0.03,
               "competition": 6, "firstSeenRange": "over_1_year" },
  "suggestEngines": { "count": 1, "active": ["google"] }
}
```

`data.query.suggestEngines` lists the engines that were queried, always as an
array even for one engine.

## related-keywords — items path `data.items`

```json
{ "keyword": "ラッコ グッズ", "metrics": { "seoDifficulty": null, "searchVolume": 1600,
                                          "cpc": 0.1, "competition": 30, "firstSeenRange": "over_1_year" } }
```

## other-keywords — items path `data.items`

```json
{ "type": "lsi", "keyword": "ラッコ 生態", "importance": "high",
  "sourceKeyword": "ラッコ", "metrics": { } }
{ "type": "paa", "question": "ラッコはなぜ絶滅危惧種なのですか？", "importance": "medium",
  "sourceKeyword": "ラッコ" }
```

One array holds both kinds. Branch on `type`: `lsi` records have `keyword` and
`metrics`, `paa` records have `question`. `data.summary` has `lsiCount` and
`paaCount` instead of `totalCount`.

## question-search — items path `data.items`

```json
{ "question": "ラッコは何を食べますか？" }
```

## ranking-keywords — items path `data.items`

```json
{ "keyword": "らっこ 生態", "wordCount": 2,
  "metrics": { "seoDifficulty": 30, "searchVolume": 720, "cpc": 0.2,
               "competition": 4, "relevance": 65 } }
```

## search-volume results — items path `data.items`

```json
{
  "keyword": "ラッコ",
  "dataSource": "google_ads",
  "metrics": { "seoDifficulty": 35, "searchVolume": 90500, "cpc": 0,
               "competition": 1, "firstSeenRange": "over_1_year" },
  "trends": {
    "changeRate": { "12m": 0.12, "6m": -0.04, "3m": 0.0,
                    "yoy1y": 0.2, "yoy2y": null, "yoy3y": null },
    "monthlySearchVolume": { "2026-01": 74000, "2026-02": 90500 }
  }
}
```

`changeRate` values are fractions: 0.12 is +12%. `yoy2y` and `yoy3y` need
`--aggregation-period-months` 36 and 48 respectively, and are null otherwise.
`monthlySearchVolume` is an object keyed `YYYY-MM`, so in CSV it lands in one
column as compact JSON — use `-f json` for trend work.

`data.query` echoes `requestId`, `location`, `language` and
`aggregationPeriodMonths`.

## search-volume status / search-rank status — no items path

```json
{ "isCompleted": false,
  "statuses": { "searchVolume": "processing", "seoDifficulty": "skip", "noiseReduction": "unprocessed" } }
```

`isCompleted` ignores `noiseReduction`; it can be true while noise reduction is
still running. For search-rank the statuses are `serp` and
`searchVolumeAndSeoDifficulty` (the latter absent when the option was off, and
`failed` / `integration_failed` keep `isCompleted` false).

## search-volume / search-rank histories — items path `data.items`

```json
{ "requestId": 1234567, "createdAt": "2026-07-25T04:00:00.000Z", "completedAt": null,
  "status": "processing", "statuses": { }, "keywordSummary": "ラッコ,カワウソ",
  "keywordCount": 2, "seoDifficulty": false, "location": "Japan", "language": "Japanese",
  "aggregationPeriodMonths": 12, "dataCompletion": true }
```

`requestId` is a **number** for search-volume and a **ULID string** for
search-rank. Sorted by `createdAt` descending, always.

## search-rank results — items path `data.items`

```json
{
  "keyword": "ラッコ",
  "metrics": { "seoDifficulty": 35, "searchVolume": 90500, "cpc": 0, "competition": 1 },
  "rankings": [
    { "target": "*.example.com/*", "position": 7,
      "rankedUrl": "https://example.com/otter", "estimatedTraffic": 120 },
    { "target": "*.other.com/*", "position": null, "rankedUrl": null, "estimatedTraffic": 0 }
  ]
}
```

`data.summary.targets[]` carries per-target totals and a
`rankingPositionDistribution` object (`1-3`, `4-10`, …, `101+`) when
`--with-aggregation` is set; without it `estimatedTraffic` there is 0.

## influx-keywords — items path `data.items`

```json
{ "target": "https://example.com/", "keyword": "ラッコ 生態",
  "metrics": { "seoDifficulty": 30, "searchVolume": 720, "cpc": 0.2, "competition": 4 },
  "ranking": { "position": 3, "estimatedTraffic": 180, "url": "https://example.com/otter" } }
```

`data.summary` adds `estimatedTraffic` and `keywordCount` for the whole target.

## influx-pages — items path `data.items`

```json
{ "target": "https://example.com/",
  "page": { "title": "ラッコの生態", "url": "https://example.com/otter" },
  "performance": { "rankingKeywordCount": 42, "estimatedTraffic": 3100, "trafficValue": 620 },
  "topKeyword": { "keyword": "ラッコ 生態", "position": 3,
                  "metrics": { "seoDifficulty": 30, "searchVolume": 720 } } }
```

## competitive — items path `data.items`

```json
{ "site": { "domain": "example.com", "title": "Example" },
  "metrics": { "estimatedTraffic": 51000, "trafficValue": 12000, "keywordCount": 8200,
               "pageCount": 430, "duplicateKeywordCount": 620, "duplicateRate": 0.42,
               "competitorUniqueKeywordCount": 7580, "targetUniqueKeywordCount": 1200 } }
```

## bulk-site-research — items path `data.items`

```json
{ "site": { "target": "*.example.com/*" },
  "metrics": { "estimatedTraffic": 51000, "estimatedTrafficChangeRate": 0.1,
               "keywordCount": 8200, "pageCount": 430, "trafficValue": 12000,
               "pagesWithTrafficCount": 210, "pagesWithTrafficRate": 0.49,
               "averageEstimatedTrafficPerPage": 118, "averageRankingKeywordCountPerPage": 19,
               "averageTrafficValuePerPage": 27 },
  "histories": [ { "date": "2026-07-31", "etvIndex": 100, "keywordCountIndex": 92.5, "pageCountIndex": 88 } ],
  "distributions": { "rankingPosition": { "1-3": 12, "4-10": 44, "11-20": 80,
                                          "21-50": 210, "51-100": 300, "1-10": 56, "1-20": 136, "1-30": 190 },
                     "pageTraffic": { "0": 220, "1-100": 150, "101-1000": 40,
                                      "1001-10000": 18, "10001+": 2, "1+": 210, "100+": 60, "1000+": 20 } } }
```

`items` is the same length and order as the `urls` given. The
`rankingPosition` and `pageTraffic` buckets overlap deliberately (`1-3` is also
inside `1-10`) — do not sum them.

## content-search — items path `data.items`

```json
{ "page": { "domain": "example.com", "url": "https://example.com/otter",
            "title": "ラッコの生態", "description": "…" },
  "metrics": { "estimatedTraffic": 3100, "trafficValue": 620, "rankingKeywordCount": 42 },
  "topKeyword": { "keyword": "ラッコ 生態", "wordCount": 2, "position": 3,
                  "metrics": { "seoDifficulty": 30, "searchVolume": 720 } } }
```

## site-search — items path `data.items`

```json
{ "no": 1,
  "site": { "domain": "example.com", "url": "https://example.com/", "title": "…", "description": "…" },
  "metrics": { "estimatedTraffic": 51000, "trafficValue": 12000,
               "rankingKeywordCount": 8200, "pageCount": 430 },
  "relatedContent": { "estimatedTraffic": 900, "relevanceScore": 62 } }
```

`relatedContent` is `null` unless a content filter (`filter.keyword.includes`)
was given.

## headline — items path `data.items`

```json
{ "page": { "url": "https://example.com/otter", "title": "…", "description": "…" },
  "metrics": { "position": 1, "headlineCount": 24, "wordCount": 8400 },
  "headlines": [ { "level": "h2", "text": "ラッコの生態" } ] }
```

`headlines` is an array of objects, so it appears as compact JSON in CSV and
table output. `data.summary` adds `averageHeadlineCount`, `averageWordCount`,
`minWordCount` and `maxWordCount`.

## co-occurrence — items path `data.items`

```json
{ "word": "生態",
  "metrics": { "occurrencePageCount": 38, "occurrenceTitleCount": 4, "occurrenceHeadingCount": 11,
               "siteCountTotal": 9, "siteCountHeading": 5 },
  "pageDetails": [ { "rank": 1, "title": "…", "url": "…", "count": 6,
                     "countInHeadline": 2, "countInTitle": 1,
                     "pageCount": 1, "pageCountInHeadline": 1 } ] }
```

`pageDetails` is omitted with `--details=false`.

## metadata — items paths `data.locations` / `data.languages`

```json
{ "name": "Shibuya,Tokyo,Japan", "countryIsoCode": "JP" }
{ "name": "Japanese" }
```
