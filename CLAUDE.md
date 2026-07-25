# CLAUDE.md — rakkokeyword-cli

ラッコキーワードAPIのCLI。**バイナリ名は `rakko`**、goreleaser のプロジェクト名も
`rakko`（リポジトリ名だけ `rakkokeyword-cli`）。

このCLIの本質は2つ。

1. **APIコールがユーザーのクレジット（＝お金）を消費する。** 各コマンドのヘルプに
   費用を明記し、`--dry-run` で無料確認できる状態を保つこと。
2. **レスポンスはGoの構造体に写さず、生バイトのまま扱う。** `-f json` はAPIの
   レスポンスそのもの。列名は `internal/output` がドット区切りで自動生成する。
   これによりAPIにフィールドが増えてもCLI側が黙って落とすことがない。

## 変更時の必須手順

**機能を追加した、フラグを増やした、既存の挙動を変えた — このいずれかをしたら、
3か所すべてを更新してから終わること。**

| 更新先 | 対象 | やり方 |
| --- | --- | --- |
| ① ドキュメント | `README.md` / `README_ja.md` | 使い方・費用が変わったときのみ |
| ② ヘルプ | cobra の `Short` / `Long` / `Example` / フラグ説明 | コード内。**カタログはここから生成される** |
| ③ **LLMナレッジ** | `internal/llmdocs/00-guide.md` | 鉄則・認証・コマンド選択表が変わったら |
| | `internal/llmdocs/10-metrics.md` | 指標の意味・単位・鮮度が変わったら |
| | `internal/llmdocs/20-schemas.md` | **JSON出力の形が変わったら必ず** |
| | `internal/llmdocs/30-gotchas.md` | 費用・上限値・非同期の挙動が変わったら |
| | `internal/llmdocs/90-commands.md` | **生成物。手編集しない** → `go generate ./...` |
| | `plugins/rakkokeyword-cli/skills/*/SKILL.md` | 手順や前提が変わったとき |
| | `context7.json` の `rules` | 新しい落とし穴が生まれたとき |

③ を忘れやすい。ドキュメントとヘルプは人間が読んで気づくが、**LLMナレッジが
古いことには誰も気づかない**（エージェントが黙って間違えるだけ）。

判断に迷ったときの目安:

- **エンドポイントを追加した** → コマンド定義（②）→ `go generate` →
  `20-schemas.md` にレスポンス形と items パス →`00-guide.md` のコマンド選択表 →
  費用を `30-gotchas.md` の表へ
- **クレジット単価が変わった** → `Short`/`Long` の「Cost:」、`call.credits`、
  README両方の費用表、`30-gotchas.md` の費用表。**5か所ある**。
  SKILL.md には費用表を置いていない（`--help` と `--dry-run` を引かせる方針）が、
  極端な3件（`other-keywords` 22.5 / `search-volume` 最低15 / `--seo-difficulty` 25倍）
  だけは書いてあるので、そこが動いたら追随する
- **フィルタのパスが増えた** → `internal/rakko/params.go` の該当 `FilterSpec`。
  ヘルプ文言は `Usage()` が自動生成する
- **上限値（件数・ターゲット数）が変わった** → コマンド内のローカル検証と
  `30-gotchas.md` の上限表
- **出力形式・列の決め方を変えた** → `20-schemas.md` と `00-guide.md` の出力契約

## クレジットに関わる変更は特に慎重に

`--dry-run` の `cost` 文字列と実際の課金がズレると、エージェントがユーザーに
嘘の見積もりを伝える。単価を変えたら `call.credits` と、`search-volume` /
`search-rank` / `bulk-site-research` の計算式（キーワード数・URL数・depth依存）を
必ず両方確認すること。

## リリース

`PluginVersion`（`cmd/rakko/main.go`）と `plugin.json` の `version` と git タグの
3つを揃える。テストとリリースワークフローが不一致を検出する。手順は
`plugins/rakkokeyword-cli/PUBLISH.md`。

## 確認

```bash
go generate ./...     # 生成物を作り直す
git diff --exit-code  # 差分が出たらコミット漏れ
go test ./...         # SKILL.md 検証とバージョン整合を含む
go run ./cmd/rakko llm | head
go run ./cmd/rakko metadata languages   # 無料・APIキー不要の疎通確認
```

APIを叩くテストは `httptest` のスタブに対して行う（`cmd/rakko/cli_test.go`）。
実APIでの確認は無料エンドポイント（`metadata`）か `--dry-run` を使い、
有料エンドポイントを検証目的で連打しないこと。

## 参照

- API仕様: <https://api.rakkokeyword.com/api-docs.json>
- 標準: <https://github.com/ideamans/go-llm-cli-kit/blob/main/LLM.md>
- 生成物と原本の対応: `.claude/rules/ai-artifacts-policy.md`
- 再生成: `/regen-ai`
