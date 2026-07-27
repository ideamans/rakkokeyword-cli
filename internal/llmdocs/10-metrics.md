# 指標とその読み方

キーワード系のレスポンスはすべて同じ `metrics` ブロックを持つ。
単位と鮮度を正しく押さえることが、このデータをうまく使うことのほとんどを占める。

| フィールド | 意味 | 値域 | 注意 |
| --- | --- | --- | --- |
| `searchVolume` | 集計期間で平均した月間検索数 | 0以上 | 不明なら `null`。Google Ads がバケット値に丸めている（10、90、480、1600、18100、90500…）ので、実測値ではなく桁として扱う |
| `seoDifficulty` | 上位表示の難しさ | 1〜100 | 1〜33が低、34〜66が中、67〜100が高。**しばしば `null`** — `search-volume --seo-difficulty` で明示的に要求したときだけ計測される |
| `cpc` | 推定クリック単価 | 0以上 | **米ドル**。円ではない |
| `competition` | リスティング広告の競合度 | 0〜100 | 0〜33が低、34〜66が中、67〜100が高。これは*広告*のシグナルであってSEOのものではない |
| `firstSeenRange` | そのキーワードがラッコキーワードのDBに最初に現れた時期 | ラベル | `last_7_days` 〜 `over_1_year`。検索数が多いのに新しければ、トレンドの可能性がある |
| `relevance` | 元キーワードとの検索意図の重なり（`ranking-keywords`） | 1〜100 | 高ければ1記事で両方を狙える |
| `importance` | LSI/PAA 項目が繰り返し現れた度合い（`other-keywords`） | high/medium/low | 再帰クロールから導出したもので、検索ボリューム由来ではない |

## 鮮度: 計測するコマンドと、思い出すだけのコマンド

**計測するのは `search-volume` だけ。** 他のコマンドは最後にキャッシュされた指標を
そのまま付けて返す。API のドキュメントにもコマンドごとに明記されている。

```
suggest-keywords / related-keywords / other-keywords / ranking-keywords
influx-keywords / influx-pages / content-search / site-search
        └── 指標は古い可能性がある。seoDifficulty はたいてい null
                        │
                        ▼
        search-volume register --wait   ← 現在の検索数・CPC・競合度・傾向、
                                          （オプトインで）SEO難易度
        search-rank register --wait     ← 現在の Google 順位
```

したがって誠実な進め方は、**安く広く発見してから、絞り込んだ候補を測り直す**こと。
`suggest-keywords` の難易度スコアをそのまま引用するのが、このAPIで自信満々に
間違える最も多いパターン。

## トラフィック指標

| フィールド | 意味 |
| --- | --- |
| `estimatedTraffic` / `etv` | 推定月間検索流入数 |
| `trafficValue` | 推定流入 × CPC。その流入を広告で買った場合の費用に相当。**USD** |
| `rankingKeywordCount` / `keywordCount` | そのページ／サイトが順位を持つキーワード数 |
| `pageCount` | ラッコキーワードが把握しているインデックス済みページ数 |
| `duplicateRate`（`competitive`） | キーワードの重複率。[0,1] の小数 — 0.42 は 42% |
| `pagesWithTrafficRate`（`bulk-site-research`） | 流入のあるページの割合。[0,1] |

いずれも順位×クリック率モデルからの推定値であって、アクセス解析の実測ではない。
1つの数値を予測として扱うのではなく、サイト同士を相対比較するために使う。

## bulk-site-research の指数

`histories[]` は正規化されている。`etvIndex`・`keywordCountIndex`・`pageCountIndex`
は 0〜100 で、系列の最大値が100になる。系列ごと・サイトごとに独立して計算される。
つまり**規模ではなく形**を表す。

現在の絶対値は `metrics`（`estimatedTraffic`、`keywordCount`、`pageCount`）にある。
指数をサイト間でボリュームのように比較してはいけないし、何かを掛けてもいけない。

## 順位データ

`search-rank` の結果には、対象URLごとに1エントリの `rankings` 配列が入る:

- `position` — `--depth` の範囲内にURLが現れなければ `null`。これは「上位N件に入っていない」
  という意味で、「0位」とも「データ無し」とも違う。
- `rankedUrl` — 実際に順位を取ったURL。問い合わせたURLとは限らない
  （`sub_domain` マッチングにより、同一ドメインの別ページのことがある）。
- `estimatedTraffic` — 順位と検索数からのモデル値。

`influx-keywords` の順位はクロールキャッシュ由来なので遅れることがある。
ユーザーが「*今*何位か」を訊いているときは `search-rank` を使う。

## suggestClass（suggest-keywords）

| ラベル | 意味 |
| --- | --- |
| `＋` | 直接のサジェスト |
| `＋＋` | サジェストのサジェスト |
| `＋α` | キーワードに あいうえお / abc / 123 を付けて得たサジェスト |
| `＋＋＋` | `＋＋` や `＋α` からさらに展開したもの |

`＋` が実際にユーザーが打っている語に最も近い。それ以外は
`--increase-keyword` 系の展開によるもので、後ろにいくほど推測の度合いが強くなる。
