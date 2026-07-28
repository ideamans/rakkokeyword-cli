# JSON 出力スキーマ

`-f json` は API のレスポンスをそのまま出力するので、ここに載せているのは
API 自身の構造そのもの。`-f jsonl` は下記の **items パス** が指す配列の要素を
1行1件で出力する。`-f csv` は同じ要素をドット記法の列名で出力する。

## エンベロープ

```json
{
  "result": true,
  "meta": { "consumedCredit": 1.5 },
  "data": { "query": {}, "summary": {}, "items": [] },
  "errors": []
}
```

`data.summary.totalCount` は条件に一致した件数、`returnedCount` は limit の範囲で
実際に返ってきた件数。両者が食い違っていれば結果は切り詰められている。
一部を全体として報告せず、`-n` を上げるかフィルタを絞ること。

## suggest-keywords — items パス `data.items`

```json
{
  "keyword": "ラッコ 水族館",
  "suggestClass": "＋",
  "metrics": { "seoDifficulty": 36, "searchVolume": 18100, "cpc": 0.03,
               "competition": 6, "firstSeenRange": "over_1_year" },
  "suggestEngines": { "count": 1, "active": ["google"] }
}
```

`data.query.suggestEngines` は問い合わせた検索エンジンの一覧。1つでも必ず配列。

## related-keywords — items パス `data.items`

```json
{ "keyword": "ラッコ グッズ", "metrics": { "seoDifficulty": null, "searchVolume": 1600,
                                          "cpc": 0.1, "competition": 30, "firstSeenRange": "over_1_year" } }
```

## other-keywords — items パス `data.items`

```json
{ "type": "lsi", "keyword": "ラッコ 生態", "importance": "high",
  "sourceKeyword": "ラッコ", "metrics": { } }
{ "type": "paa", "question": "ラッコはなぜ絶滅危惧種なのですか？", "importance": "medium",
  "sourceKeyword": "ラッコ" }
```

1つの配列に2種類が混在する。`type` で分岐すること。`lsi` は `keyword` と `metrics` を、
`paa` は `question` を持つ。`data.summary` には `totalCount` ではなく
`lsiCount` と `paaCount` が入る。

## question-search — items パス `data.items`

```json
{ "question": "ラッコは何を食べますか？" }
```

## ranking-keywords — items パス `data.items`

```json
{ "keyword": "らっこ 生態", "wordCount": 2,
  "metrics": { "seoDifficulty": 30, "searchVolume": 720, "cpc": 0.2,
               "competition": 4, "relevance": 65 } }
```

## search-volume results — items パス `data.items`

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

`changeRate` は小数。0.12 は +12% を意味する。`yoy2y` と `yoy3y` はそれぞれ
`--aggregation-period-months` に 36 と 48 が必要で、無ければ null。
`monthlySearchVolume` は `YYYY-MM` をキーにしたオブジェクトなので、csv では
コンパクトなJSONとして1列に収まる。傾向分析には `-f json` を使う。

`data.query` には `requestId`・`location`・`language`・`aggregationPeriodMonths`
がそのまま返る。

## search-volume status / search-rank status — items パスなし

```json
{ "isCompleted": false,
  "statuses": { "searchVolume": "processing", "seoDifficulty": "skip", "noiseReduction": "unprocessed" } }
```

`isCompleted` は `noiseReduction` を無視するため、ノイズ除去が動いている最中でも
true になりうる。search-rank のステータスは `serp` と
`searchVolumeAndSeoDifficulty`（後者はオプション未指定なら存在しない。
`failed` / `integration_failed` の間は `isCompleted` が false のまま）。

## search-volume / search-rank histories — items パス `data.items`

```json
{ "requestId": 1234567, "createdAt": "2026-07-25T04:00:00.000Z", "completedAt": null,
  "status": "processing", "statuses": { }, "keywordSummary": "ラッコ,カワウソ",
  "keywordCount": 2, "seoDifficulty": false, "location": "Japan", "language": "Japanese",
  "aggregationPeriodMonths": 12, "dataCompletion": true }
```

`requestId` は search-volume では**数値**、search-rank では **ULID文字列**。
並び順は常に `createdAt` の降順。

## search-rank results — items パス `data.items`

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

`data.summary.targets[]` にはターゲットごとの合計と、`--with-aggregation` 指定時に
`rankingPositionDistribution` オブジェクト（`1-3`、`4-10`、…、`101+`）が入る。
未指定の場合そこの `estimatedTraffic` は 0 になる。

## influx-keywords — items パス `data.items`

```json
{ "target": "https://example.com/", "keyword": "ラッコ 生態",
  "metrics": { "seoDifficulty": 30, "searchVolume": 720, "cpc": 0.2, "competition": 4 },
  "ranking": { "position": 3, "estimatedTraffic": 180, "url": "https://example.com/otter" } }
```

`data.summary` にはターゲット全体の `estimatedTraffic` と `keywordCount` が加わる。

## influx-pages — items パス `data.items`

```json
{ "target": "https://example.com/",
  "page": { "title": "ラッコの生態", "url": "https://example.com/otter" },
  "performance": { "rankingKeywordCount": 42, "estimatedTraffic": 3100, "trafficValue": 620 },
  "topKeyword": { "keyword": "ラッコ 生態", "position": 3,
                  "metrics": { "seoDifficulty": 30, "searchVolume": 720 } } }
```

## competitive — items パス `data.items`

```json
{ "site": { "domain": "example.com", "title": "Example" },
  "metrics": { "estimatedTraffic": 51000, "trafficValue": 12000, "keywordCount": 8200,
               "pageCount": 430, "duplicateKeywordCount": 620, "duplicateRate": 0.42,
               "competitorUniqueKeywordCount": 7580, "targetUniqueKeywordCount": 1200 } }
```

## bulk-site-research — items パス `data.items`

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

`items` の件数と並び順は、渡した `urls` と同じ。`rankingPosition` と `pageTraffic`
のバケットは意図的に重複している（`1-3` は `1-10` にも含まれる）ので、合算してはいけない。

## content-search — items パス `data.items`

```json
{ "page": { "domain": "example.com", "url": "https://example.com/otter",
            "title": "ラッコの生態", "description": "…" },
  "metrics": { "estimatedTraffic": 3100, "trafficValue": 620, "rankingKeywordCount": 42 },
  "topKeyword": { "keyword": "ラッコ 生態", "wordCount": 2, "position": 3,
                  "metrics": { "seoDifficulty": 30, "searchVolume": 720 } } }
```

## site-search — items パス `data.items`

```json
{ "no": 1,
  "site": { "domain": "example.com", "url": "https://example.com/", "title": "…", "description": "…" },
  "metrics": { "estimatedTraffic": 51000, "trafficValue": 12000,
               "rankingKeywordCount": 8200, "pageCount": 430 },
  "relatedContent": { "estimatedTraffic": 900, "relevanceScore": 62 } }
```

`relatedContent` は、コンテンツフィルタ（`filter.keyword.includes`）を指定しない限り
`null`。

## headline — items パス `data.items`

```json
{ "page": { "url": "https://example.com/otter", "title": "…", "description": "…" },
  "metrics": { "position": 1, "headlineCount": 24, "wordCount": 8400 },
  "headlines": [ { "level": "h2", "text": "ラッコの生態" } ] }
```

`headlines` はオブジェクトの配列なので、csv と table ではコンパクトなJSONとして現れる。
`data.summary` には `averageHeadlineCount`・`averageWordCount`・`minWordCount`・
`maxWordCount` が加わる。

## co-occurrence — items パス `data.items`

```json
{ "word": "生態",
  "metrics": { "occurrencePageCount": 38, "occurrenceTitleCount": 4, "occurrenceHeadingCount": 11,
               "siteCountTotal": 9, "siteCountHeading": 5 },
  "pageDetails": [ { "rank": 1, "title": "…", "url": "…", "count": 6,
                     "countInHeadline": 2, "countInTitle": 1,
                     "pageCount": 1, "pageCountInHeadline": 1 } ] }
```

`pageDetails` は `--details=false` で省かれる。

## metadata — items パス `data.locations` / `data.languages`

```json
{ "name": "Shibuya,Tokyo,Japan", "countryIsoCode": "JP" }
{ "name": "Japanese" }
```
