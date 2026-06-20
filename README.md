# schritt

PBI を起点に、**リファインメント → 実装計画レビュー(pre-exec review) → 実装 → 検証 → PR作成**
を段階的に進めるための、Go + [Bubble Tea] 製の TUI ツールです。

> *schritt* — ドイツ語で「一歩 / ステップ」。

現在実装済みなのは最初のステップ、**リファインメント**です。

## リファインメント

```sh
schritt refinement                    # claude で実行（既定）
schritt refinement --engine codex     # OpenAI codex で実行
schritt refinement --demo             # AIを呼ばずサンプル結果で挙動を確認
```

AIエンジンは `--engine claude|codex` で切り替えられます（既定は `claude`）。
`--model` でモデル名、`--bin` で実行ファイルのパスを上書きできます。

> **事前に skill のインストールが必要です。** schritt は AI を「素のプロンプト」で
> 呼ぶのではなく、**refine-pbi skill を名前で起動**します（後述）。初回は
> `make install-skills`（claude）/ `scripts/install-codex.sh`（codex）を実行してください。
> `--demo` は skill 不要で動きます。

### フロー

1. 起動すると **PBI入力画面** が開きます。
   - `PBI #` に PBI 番号を入力
   - `tab` で本文欄に移り、PBI のマークダウンを貼り付け
   - `tab` で **補足欄** に移り、リファインメント会議で話した内容・前提・決定事項などを記入（任意）
2. `Ctrl+R` で **AI によるリファインメント** を実行します（refine-pbi skill を起動）。
   補足は PBI 本文と合わせて AI のコンテキストに渡されます（会議での決定事項を優先するよう指示）。
3. 完了すると **結果画面** に切り替わり、次の4セクションを確認できます。
   - POへの確認事項
   - 実装内容
   - 単体テストのテストケース
   - 統合テストのテストケース

### キーバインド

| Key            | 画面        | Action                          |
| -------------- | ----------- | ------------------------------- |
| `tab`          | PBI入力     | PBI番号 / 本文 / 補足 のフィールド切替 |
| `Ctrl+R`       | PBI入力     | リファインメント実行            |
| `j` / `k`      | 結果        | セクション移動 / 行スクロール   |
| `Enter`        | 結果(一覧)  | セクションを開く                |
| `l` / `Esc`    | 結果(詳細)  | 一覧へ戻る                      |
| `:`            | 結果        | コマンド (`:new` `:q` `:help`)  |
| `?`            | 結果        | ヘルプの表示/非表示             |
| `Ctrl+C`       | 全画面      | 終了                            |

### 出力フォーマット

revu に倣い、メタ情報を YAML、本文を markdown ファイルで管理します。

```
~/.schritt/pbi-{番号}/{日時}/
  refinement.yml        メタ情報 + 各セクションへの参照
  pbi.md                入力したPBI（参照用に保存）
  notes.md              入力した補足メモ（あれば／参照用に保存）
  po_questions.md       POへの確認事項
  implementation.md     実装内容
  unit_tests.md         単体テストのテストケース
  integration_tests.md  統合テストのテストケース
```

`refinement.yml` の例:

```yaml
schema_version: 1
pbi:
    number: 42
    title: ログイン機能
generated_at: 2026-06-19T04:41:31Z
generated_by:
    tool: schritt
    model: demo
sections:
    - id: po_questions
      title: POへの確認事項
      body_file: po_questions.md
    - ...
```

> `~/.schritt` の場所は `SCHRITT_HOME` 環境変数で上書きできます（主にテスト用）。

### AI の呼び出しについて（skill）

AIへの指示は **`skills/refine-pbi/SKILL.md` という1つの skill** に切り出してあり、
これを各ランタイムの skill ディレクトリにインストールして **名前で起動** します。
revu が1つの `review-pr` skill を claude / codex 双方から呼ぶのと同じ方式です。

| ランタイム | skill ディレクトリ | 起動構文 | schritt の起動 |
| ---------- | ------------------ | -------- | -------------- |
| Claude Code | `~/.claude/skills/` | `/refine-pbi <dir>` | `claude --print "/refine-pbi <dir>"` |
| OpenAI Codex CLI | `~/.agents/skills/` | `$refine-pbi <dir>` | `codex exec … "$refine-pbi <dir>"` |

`schritt refinement` は次の流れで動きます（`internal/refine/`）。

1. PBI本文（と補足があれば `notes.md`）を一時作業ディレクトリ `<dir>` に書き出す
2. 上表の構文で **skill を名前で起動**（`<dir>` を引数で渡す）
3. skill が `<dir>` に書いた4つのマークダウンを読み戻し、`~/.schritt/...` に
   `refinement.yml` + markdown として整形・保存する

skill 本体は1つなので、指示を変えたいときは `SKILL.md` を編集するだけです。
エンジンごとの差分は起動引数だけで、`internal/refine/claude.go` / `codex.go` に分離しています。

#### skill のインストール

```sh
# Claude Code 用: skills/* を ~/.claude/skills/ にシンボリックリンク
make install-skills
make uninstall-skills

# Codex CLI 用: skills/refine-pbi を ~/.agents/skills/ にリンク + codex 再起動
make install-codex-skills      # = scripts/install-codex.sh
make uninstall-codex-skills
```

> Codex はファイル単位の symlink を落とす ([openai/codex#15756]) ため、
> **ディレクトリごと** リンクします。スクリプトがその処理を行います。

#### チャットインターフェースから使う

インストール後は **どちらのチャットからも** 同じ skill を使えます。

- Claude Code: PBIを貼り付けて「リファインメントして」と頼むか、`/refine-pbi` を起動
- Codex CLI: `$refine-pbi` を起動（`--copy` でインストールした場合も同様）

skill の `description` を見て自動的に選択されます。`<dir>` を渡さずチャットで直接
PBIを貼った場合は、4セクションを返信としてそのまま出力します（SKILL.md に明記）。

AI を使わずに UI を試したい場合は `--demo` を使ってください（skill 不要）。

## 開発

```sh
make run ARGS="refinement --demo"   # build + 起動
make build                          # bin/schritt をビルド
make test                           # テスト
make lint                           # golangci-lint
```

## レイアウト

```
cmd/schritt/main.go      Cobra エントリポイント。refinement サブコマンドを定義。
internal/model/          ドメイン型（Refinement / Section / PBIMeta）。
internal/refine/         AI境界。Refiner interface、共通ランナー(cli.go)、
                         ClaudeRefiner / CodexRefiner / DemoRefiner。skill を名前で起動。
internal/store/          refinement.yml + markdown の保存・読み込み。
internal/tui/app.go      ルートの Bubble Tea モデル（状態遷移・コマンド・ヘルプ）。
internal/tui/views/      画面（input / list / detail）と画面遷移メッセージ。
internal/tui/keys/       キーバインドの一元管理。
skills/refine-pbi/       refine-pbi skill (SKILL.md)。claude/codex 双方で共有する単一ソース。
scripts/install-codex.sh Codex CLI 用 skill インストーラ (~/.agents/skills)。
```

`refine.Refiner` を差し替え口にしているため、AI 呼び出し方法を変えても TUI / store
層は影響を受けません。テストや `--demo` は `DemoRefiner` を使います。

## 今後のステップ（予定）

リファインメントに続けて、`pre-exec review` → 実装 → 検証 → PR作成 のサブコマンドを
同じ `cmd` / `internal/tui` / `internal/store` の構造に沿って追加していく想定です。

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Claude Code]: https://docs.claude.com/en/docs/claude-code
[Codex CLI]: https://github.com/openai/codex
[Codex skills]: https://developers.openai.com/codex/skills
[Claude Code skill]: https://docs.claude.com/en/docs/claude-code/skills
[openai/codex#15756]: https://github.com/openai/codex/issues/15756
