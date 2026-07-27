# rakkokeyword — AIエージェント向けリファレンス

`rakkokeyword` はラッコキーワード API の CLI。キーワード調査、SEO指標、検索順位、
競合・コンテンツ分析を扱う。対象は主に日本語検索市場。

## このリファレンスの読み方

**全部読まないこと。** 全体で約1,450行あり、その大半はコマンドカタログ。
通読する価値があるのはこの最初の章だけで、残りは引くもの。

| 章 | 読むタイミング |
| --- | --- |
| `00-guide.md`（この章） | 常に。鉄則・認証・どのコマンドが何に答えるか |
| `10-metrics.md` | 数値を報告する前に。単位・値域・鮮度 |
| `20-schemas.md` | パースする前に。各レスポンスのJSON構造 |
| `30-gotchas.md` | 大量・反復呼び出しの前に。コスト・制限・非同期の挙動 |
| `90-commands.md` | ほとんど不要。1コマンドなら `rakkokeyword <command> --help` が同じことを言う |

リファレンス全体ではなく、必要な章だけ取り出す:

```bash
rakkokeyword llm --format json | jq -r '.[] | select(.file=="20-schemas.md") | .body'
rakkokeyword llm | sed -n '/^# 制限・コスト・落とし穴/,$p'
rakkokeyword suggest-keywords --help          # 1コマンドのフラグ・既定値・コスト
```

## 四つの鉄則

1. **すべての呼び出しがユーザーの金を消費する。** クレジットは API キーが属する
   アカウントから引かれる。コストは無料（metadata / histories / status / results）、
   1.5クレジット（suggest / related / site-search）から、`other-keywords` 1回の
   22.5クレジットまで幅がある。実行前にコストを伝えること。`--dry-run` を使えば
   クレジットを消費せずにリクエスト内容を確認できる。
2. **パースするなら必ず `-f json` を付ける。** 既定の `table` は人間向けで、
   長い値を切り詰め、列も一部しか出さない。`-f json` は API レスポンスをそのまま
   出力し、`-f jsonl` は1レコード1行、`-f csv` は全フィールドをドット記法の列に平坦化する。
3. **`null` は「不明」であって 0 ではない。** `seoDifficulty` はしばしば null になり、
   順位結果の `position` が null なら「`--depth` の範囲内に見つからなかった」という意味。
   どちらも 0 として報告してはいけない。
4. **バッチ系コマンドは非同期。** `search-volume` と `search-rank` は
   register → status → results の3段構え。理由がない限り `register --wait` を使う。

## 認証

API キーの解決順序:

1. `--api-key`
2. `RAKKOKEYWORD_API_KEY`（推奨。ディスクには何も書かれない）
3. `RAKKO_API_KEY`
4. 設定ファイル（`rakkokeyword auth set-api-key <key>`）

`rakkokeyword auth status` はキー自体を晒さずに、どの経路が使われているかを表示する。
キーはラッコキーワードの STANDARD プラン以上で発行できる（1アカウントにつき最大5本）。
キー無しで動くのは `rakkokeyword metadata locations` と
`rakkokeyword metadata languages` の2つだけ。

## どのコマンドがどの問いに答えるか

| ユーザーが知りたいこと | コマンド | コスト |
| --- | --- | --- |
| 広く関連するキーワード、実際のサジェスト | `suggest-keywords` | 1.5 |
| ある語を含むキーワードを数万件 | `related-keywords` | 1.5 |
| 「その次に」検索・質問されること（LSI / PAA） | `other-keywords` | 22.5 |
| FAQ・AI検索向けの質問形 | `question-search` | 3 |
| 検索意図が同じキーワード群 | `ranking-keywords` | 4.5 |
| リストの正確・最新の検索ボリューム／難易度 | `search-volume register --wait` | 1キーワード 0.03〜0.78、最低15 |
| あるサイトの現在の順位 | `search-rank register --wait` | 1キーワード 0.9〜 |
| サイト（や競合）が獲得しているキーワード | `influx-keywords` | 4.5 |
| サイトのどのページが流入を稼いでいるか | `influx-pages` | 4.5 |
| SEO上の競合は誰か | `competitive` | 4.5 |
| 多数のサイトの規模と傾向を一括で | `bulk-site-research` | 1URL 0.45、最低4.5 |
| あるトピックを扱うページ（寄稿先・調査） | `content-search` | 4.5 |
| あるトピックを扱うサイト | `site-search` | 1.5 |
| 上位記事に必要な見出し | `headline` | 3 |
| 上位記事に必要な語彙 | `co-occurrence` | 3 |
| `--location` / `--language` に指定できる値 | `metadata locations` / `metadata languages` | 無料 |
| 上記でラップしていない操作 | `raw <METHOD> <path>` | そのエンドポイントのコスト |

## 出力の契約

レスポンスは共通のエンベロープを持つ:

```json
{ "result": true, "meta": { "consumedCredit": 1.5 }, "data": { }, "errors": [] }
```

- `-f json` はこのエンベロープ全体をそのまま出力する。
- `-f jsonl` と `-f csv` は `data.items`（metadata では `data.locations` /
  `data.languages`）配下のレコードを出力する。
- クレジット消費量と進捗メッセージは **stderr** に出るので、stdout はパース可能なまま。
- 2xx 以外はエラーとなり終了コードが非0になる。API 自身のメッセージも含まれる。

## 列の選び方

table と csv の列はレコードへのドット記法パス — `metrics.searchVolume`、
`ranking.position`、`page.url`。`--fields` で上書きできる:

```bash
rakkokeyword suggest-keywords ラッコ --fields keyword,metrics.searchVolume -f csv
```

table は厳選した一部の列だけを出す。csv はレスポンスが持つ全フィールドを出す。

## 絞り込み

`--filter path=value` を繰り返し指定する。指定できるパスは各コマンドの `--help` にある。
リスト値のパスを繰り返すと追加になる。

```bash
rakkokeyword related-keywords ラッコ \
  --filter searchVolume.min=100 --filter searchVolume.max=10000 \
  --filter keyword.notIncludes=グッズ,中古
```

未知のパスはクレジットを消費する前にローカルで弾かれる。`--filter` で表現できない
ものには、生の JSON フィルタオブジェクトを渡す `--filter-json` を使う。

## 実例: SEO記事の構成を組み立てる

```bash
# 1. 需要を把握する（1.5クレジット）
rakkokeyword suggest-keywords ラッコ --increase-keyword --filter searchVolume.min=100 -f json > suggest.json

# 2. 絞り込んだ候補の数値を確定する（最低15クレジット・約10秒）
rakkokeyword search-volume register --keywords-file shortlist.txt --wait -f json > volume.json

# 3. 上位ページが何を書いているかを見る（3 + 3クレジット）
rakkokeyword headline ラッコ -f json > headlines.json
rakkokeyword co-occurrence ラッコ --details=false -f json > vocabulary.json

# 4. 読者が実際に抱く疑問に答える（3クレジット）
rakkokeyword question-search ラッコ -n 50 -f json > questions.json
```

## 失敗モード

| 症状 | 原因 | 対処 |
| --- | --- | --- |
| `no API key` | 何も設定されていない | `export RAKKOKEYWORD_API_KEY=…` または `rakkokeyword auth set-api-key` |
| HTTP 403 | キーが誤りか失効 | `rakkokeyword auth status` を確認し、キーを再発行する |
| HTTP 402 | クレジット残高切れ | CLI 側では何もできない。ユーザーに補充してもらう |
| HTTP 429 | レート制限 | 既にバックオフ付きで再試行済み。呼び出し間隔を空けるか並列度を下げる |
| HTTP 400 | API が受け付けないパラメータ | そのコマンドの `--help` を読み直す。メッセージが該当フィールドを示す |
| `unknown filter path` | そのエンドポイントで無効なフィルタ | エラーが受け付けるパスを列挙している |
| status が完了しない | キューが混雑 | ジョブは数時間かかることがある。requestId は有効なままなので後で取得する |
| `data.items` が空 | フィルタが厳しすぎるかデータが無い | サイトやキーワードに何も無いと結論する前にフィルタを緩める |
