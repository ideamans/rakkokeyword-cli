# rakkokeyword-cli

[ラッコキーワードAPI](https://api.rakkokeyword.com/api-docs.json) のコマンドライン
クライアントです。キーワードリサーチ、SEO指標、検索順位チェック、競合調査、
記事構成の下調べまでを1つのコマンドで扱えます。

APIの全機能に対応し、出力は `table` / `json` / `jsonl` / `csv`、AIエージェント向けの
リファレンスは `rakko llm` としてバイナリに埋め込んであります。

```bash
rakko suggest-keywords ラッコ -n 5
```

```
consumed 1.5 credit(s)
keyword=ラッコ  totalCount=864  returnedCount=5
   keyword        suggestClass   metrics.searchVolume   metrics.seoDifficulty   metrics.cpc
------------------+------------+----------------------+-----------------------+-------------
  ラッコ            ＋                          90500                      35             0
  らっこ            ＋                          90500                      32             0
  ラッコキーワード   ＋                          49500                      40          6.43
```

## インストール

```bash
go install github.com/ideamans/rakkokeyword-cli/cmd/rakko@latest
```

または [リリースページ](https://github.com/ideamans/rakkokeyword-cli/releases) から
バイナリを取得します。アーカイブ名はリポジトリ名ではなくバイナリ名基準で
`rakko_<version>_<os>_<arch>.tar.gz` です。

## 認証

APIキーはラッコキーワードのスタンダードプラン以上で発行できます（最大5個）。

```bash
export RAKKOKEYWORD_API_KEY=your-key   # 推奨。ディスクに何も残さない
rakko auth set-api-key your-key        # 設定ファイルに保存する場合
rakko auth status                      # キー自体は表示せず、取得元だけ表示
```

優先順位は `--api-key` → `RAKKOKEYWORD_API_KEY` → `RAKKO_API_KEY` → 設定ファイル
（`~/.config/rakkokeyword-cli/config.json`）です。

`rakko metadata locations` と `rakko metadata languages` はAPIキーなしで動きます。

## クレジット消費

APIコールはアカウントのクレジットを消費します。各コマンドは `--help` に費用を明記し、
実行後は消費量をstderrに出力し、`--dry-run` なら1クレジットも使わずにリクエスト内容と
費用を確認できます。

```bash
rakko other-keywords ラッコ --dry-run
```

| コマンド | 消費 |
| --- | --- |
| `suggest-keywords` / `related-keywords` / `site-search` | 1.5 |
| `question-search` / `headline` / `co-occurrence` | 3 |
| `ranking-keywords` / `influx-keywords` / `influx-pages` / `competitive` / `content-search` | 4.5 |
| `other-keywords` | 22.5 |
| `bulk-site-research` | 1URLあたり0.45（最低4.5） |
| `search-volume register` | 1KWあたり0.03（`--seo-difficulty` で+0.75）、最低15 |
| `search-rank register` | 1KWあたり0.9（1〜30位）、取得範囲10位追加ごとに+0.3 |
| `metadata` / `histories` / `status` / `results` | 無料 |

とくに注意すべき点:

- **一括系はまとめて投げる。** `search-volume register` は1KWでも最低15クレジット。
- **`--seo-difficulty` は25倍**。しかも最大60分かかるため、SEO難易度が目的でなければOFF。
- **`other-keywords` は1回22.5クレジット**。キーワードリストでループしないこと。

## コマンド

### キーワード発掘

```bash
rakko suggest-keywords ラッコ --modes google,bing --increase-keyword   # サジェスト
rakko related-keywords ラッコ --match-type phraseMatch -n 5000         # 関連KW大量取得
rakko other-keywords ラッコ                                            # LSI + 他の人はこちらも質問
rakko question-search ラッコ -n 200                                    # よくある質問
rakko ranking-keywords ラッコ --search-top 10 --search-range 20        # 同時ランクインKW
```

### 最新のSEO指標・検索順位（非同期）

```bash
rakko search-volume register --keywords-file keywords.txt --wait
rakko search-volume histories
rakko search-volume status 1234567
rakko search-volume results 1234567 -n 500 -f csv

rakko search-rank register ラッコ --url https://example.com/ --depth 100 --device mobile --wait
rakko search-rank results 01HQZX… --with-aggregation -f json
```

`register --wait` は「登録 → ステータス確認 → 結果取得」を1コマンドで実行します。
`--wait` を付けない場合も requestId が表示されるので、後から結果を取得できます
（ジョブはバックグラウンドで進み続けます）。

### サイト・競合調査

```bash
rakko influx-keywords --target https://example.com/ --match-type sub_domain
rakko influx-pages --target https://example.com/ -n 50
rakko competitive https://example.com/
rakko bulk-site-research --urls-file sites.txt
rakko content-search ラッコ --search-target title
rakko site-search --filter keyword.includes=ラッコ
```

### 記事構成の下調べ

```bash
rakko headline ラッコ                       # 上位ページの見出し(h1〜h6)
rakko co-occurrence ラッコ --details=false   # 上位ページの共起語
```

### エスケープハッチ

CLIがラップしていないパラメータでも、生のJSONで任意のエンドポイントを叩けます。

```bash
rakko raw POST /v1/suggest-keywords --data '{"keyword":"ラッコ","modes":["google"]}'
rakko raw GET /v1/metadata/locations --query countryCode=JP
```

## 出力形式

```bash
rakko suggest-keywords ラッコ                 # table（人間向け。省略あり）
rakko suggest-keywords ラッコ -f json         # APIレスポンスそのまま
rakko suggest-keywords ラッコ -f jsonl        # 1レコード1行
rakko suggest-keywords ラッコ -f csv          # 全フィールドをドット区切り列で
rakko suggest-keywords ラッコ --fields keyword,metrics.searchVolume -f csv
```

消費クレジット・進捗・警告はstderrに出るため、`> file` でデータだけを取り出せます。
既定の形式は `rakko auth set-format json` で変更できます。

## 絞り込み

`--filter path=value` を繰り返し指定します。使えるパスは各コマンドの `--help` に
一覧があり、存在しないパスはAPIを叩く前にエラーになります。

```bash
rakko related-keywords ラッコ \
  --filter searchVolume.min=100 \
  --filter searchVolume.max=10000 \
  --filter keyword.notIncludes=グッズ,中古 \
  --sort-by searchVolume --order-by desc -n 1000
```

`--filter` で表現できない条件は `--filter-json '{…}'` で生のJSONを渡せます。

## 数値の読み方（つまずきやすい点）

- **`null` は0ではない。** `seoDifficulty` が null は「未計測」、順位の `position` が
  null は「`--depth` 以内に見つからなかった」という意味です。
- **`cpc` と `trafficValue` はUSD**です。
- **月間検索数はGoogle広告のバケット値**（90 / 480 / 1600 / 18100 / 90500 …）。
  同じ値のKWは「同じ規模帯」であって同順ではありません。
- **SEO指標が最新なのは `search-volume` だけ**。他のコマンドが返す指標はキャッシュで、
  重要な判断に使うなら `search-volume register --wait` で取り直します。
- **`duplicateRate` などの比率は0〜1**。100倍してからパーセント表記にします。
- **`bulk-site-research` の histories は0〜100の指数**で、流入数の実数ではありません。
- **`totalCount` と `returnedCount`** が違うときは件数上限で切られています。

## AIエージェント向け

```bash
rakko llm                  # 全リファレンス（鉄則・指標・スキーマ・コマンドカタログ）
rakko llm --format json    # 同じ内容を章ごとのJSONで
```

リファレンスはバイナリに埋め込まれているためオフラインでも動き、実行中のバージョンと
必ず一致します。あわせてClaude Codeプラグイン（`plugins/rakkokeyword-cli`）と
context7用の `context7.json` を同梱しています。プラグインのスキルは
`gh skill install` でCopilot / Cursor / Gemini CLIにも導入できます。

## 開発

```bash
go generate ./...     # internal/llmdocs/90-commands.md を再生成
go test ./...         # SKILL.md 検証・バージョン整合を含む
git diff --exit-code  # 生成物が古いままならCIが落ちる
```

## ライセンス

MIT © アイデアマンズ株式会社
