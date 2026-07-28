# 制限・コスト・落とし穴

## クレジット

| コマンド | コスト |
| --- | --- |
| `suggest-keywords`, `related-keywords`, `site-search` | 1リクエスト 1.5 |
| `question-search`, `headline`, `co-occurrence` | 1リクエスト 3 |
| `ranking-keywords`, `influx-keywords`, `influx-pages`, `competitive`, `content-search` | 1リクエスト 4.5 |
| `other-keywords` | **1リクエスト 22.5** |
| `bulk-site-research` | 1URL 0.45、最低 4.5 |
| `search-volume register` | 1キーワード 0.03、`--seo-difficulty` 付きは **1キーワードあたり +0.75**、1リクエスト最低 15 |
| `search-rank register` | 1〜30位は1キーワード 0.9、`--depth` を10位増やすごとに1キーワードあたり +0.3 |
| `metadata`, `histories`, `status`, `results` | 無料 |

行動を変えるべき帰結:

- **バッチ系はまとめて投げる。** `search-volume register` は1キーワードでも
  500キーワードでも最低15クレジットかかる。先に候補を集めきること。
- **`--seo-difficulty` は単価が25倍**になり、1時間かかることもある。
  難易度そのものが問いでない限り付けない。
- **`other-keywords` 1回は `suggest-keywords` 15回分。** キーワードのリストに対して
  ループで回してはいけない。
- **`--dry-run` は無料**で、送信するリクエストとその価格をそのまま表示する。
- 消費クレジットは毎回 stderr に出力され、`-f json` では `meta.consumedCredit` に入る。

## ハードリミット

| 制限 | 対象 |
| --- | --- |
| 20ターゲット | `influx-keywords`, `influx-pages` |
| 50URL | `search-rank register --url` |
| 100URL | `bulk-site-research`（STANDARD プラン以上） |
| 100レコード | `site-search`、および `histories` の1ページあたり |
| 200レコード | `question-search` |
| 5,000レコード | `ranking-keywords`, `content-search` |
| 10,000レコード | `influx-keywords`, `influx-pages` |
| 25,000レコード | `related-keywords` |
| 50,000キーワード | `search-volume register` |
| offset + limit ≤ 50,000 | `histories` |

ターゲット数・URL数の超過は CLI がローカルで弾く。それ以外は HTTP 400 で返る。

## 非同期ジョブ

```
register  →  requestId  →  status（ポーリング）  →  results
```

- ポーリング間隔は30秒より短くしない。`--wait` の既定値がそれ。
- `search-volume` の完了は通常10秒程度、`--seo-difficulty` 付きだと最大60分。
  `search-rank` は10キーワード以下なら数分、それを超えると最大1時間。
  キューが混んでいればどちらも数時間かかりうる。
- `--wait-timeout`（既定1時間）は CLI が待つのをやめるだけで、ジョブは動き続ける。
  `rakkokeyword search-volume histories` で拾い直し、完了後に requestId で結果を取得する。
- 完了前に results を取ると、エラーにならず部分的なデータが返る。
  先に `isCompleted` を確認すること。
- `isCompleted` は search-volume では `noiseReduction` を無視する。search-rank では
  metrics 段階が `failed` / `integration_failed` だと、SERP順位が取れていても false のままになる。

## データの落とし穴

- **`null` ≠ 0。** `seoDifficulty` の null は「未計測」。`position` の null は
  「`--depth` の範囲内に見つからなかった」。
- **`cpc` と `trafficValue` は USD。** 日本市場向けのデータであっても例外ではない。
- **検索ボリュームは Google Ads のバケット値**（90、480、1600、18100、90500…）。
  同じ値の2語は同じ帯にいるというだけで、人気が等しいわけではない。
- **`duplicateRate` と `pagesWithTrafficRate` は [0,1] の小数。**
  パーセント表記にするなら100倍する。
- **`bulk-site-research` の histories は 0〜100 の指数**であって、トラフィック量ではない。
- **`totalCount` と `returnedCount` を比べる。** limit は黙って切り詰めるので、
  「このサイトは100キーワードで順位を取っている」と言う前に必ず突き合わせる。
- **`site-search` はコンテンツフィルタ併用時、100件を超えてページングできない。**
  API が先に上位100サイトを選び、その後でフィルタするため。
- **`ranking-keywords --search-top` / `--search-range` は結果の意味を変える。**
  範囲を広く取ると、関連の薄いキーワードが出るのは仕様。
- **`search-rank` のマッチング。** 既定の `sub_domain` はあらゆるサブドメインを数える。
  実際にどのページが順位を取ったかは `rankedUrl` を見る。

## 出力の落とし穴

- table 出力はセルを44文字で切り詰め（`--wide` で無効化）、列も厳選した一部だけを出す。
  スクレイピングしてはいけない。`-f json` / `-f jsonl` / `-f csv` を使う。
- csv の列は API 自身のフィールド順のドット記法パス。オブジェクトの入れ子配列
  （`headlines`、`rankings`、`pageDetails`、`histories`）は、コンパクトなJSON1列になる。
- クレジット・進捗・API の警告は stderr に出る。stdout だけをリダイレクトすれば
  データはきれいなまま。
- `--fields` はスキーマ章にあるドット記法パスをすべて受け付け、table と csv の両方に効く。

## 地域と言語

`--location` と `--language` はコードではなく名前を取る（`Japan`、`Japanese`）。
市区町村レベルは `City,Region,Country` の形で指定する（`Shibuya,Tokyo,Japan`）。
都道府県単独では動かない。`rakkokeyword metadata locations --country-code JP` で
確認できる（無料・APIキー不要）。

## CLI にパラメータが無いとき

`rakkokeyword raw <METHOD> <path> --data '{…}'` が、同じ認証・リトライ・整形のまま
任意のエンドポイントへ任意の内容を送る。API の仕様は
<https://api.rakkokeyword.com/api-docs.json>。
