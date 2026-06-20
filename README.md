# schritt

PBI を起点に、**リファインメント → 実装計画レビュー(pre-exec review) → 実装 → 検証 → PR作成**
を段階的に進めるための、Go + [Bubble Tea] 製の TUI ツールです。

> *schritt* — ドイツ語で「一歩 / ステップ」。

現在実装済みなのは **リファインメント**（ステップ1）と **実装**（ステップ3）です。

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
> `make install-plugin`（claude）/ `scripts/install-codex.sh`（codex）を実行してください。
> `--demo` は skill 不要で動きます。

### フロー

1. 起動すると **PBI入力画面** が開きます。
   - `PBI #` に PBI 番号を入力
   - `tab` で **リポジトリ欄** に移り、対象リポジトリのパスを入力（任意）。指定すると
     AI がコードベースを参照し、実装内容・テストケースを具体化します（参照専用）。
     **複数指定する場合はカンマ区切り**（例: `~/front, ~/back`）
   - `tab` で本文欄に移り、PBI のマークダウンを貼り付け
   - `tab` で **補足欄** に移り、リファインメント会議で話した内容・前提・決定事項などを記入（任意）
2. `Ctrl+R` で **AI によるリファインメント** を実行します（refine-pbi skill を起動）。
   補足は PBI 本文と合わせて AI のコンテキストに渡されます（会議での決定事項を優先するよう指示）。
3. 完了すると **結果画面** に切り替わり、次のセクションを確認できます。
   - POへの確認事項
   - 実装内容（**実装ステップごと**に分割。一覧では各ステップが個別の行になります）
   - 単体テストのテストケース
   - 統合テスト（**E2Eシナリオごと**に分割。後段の動作確認が1シナリオ＝1検証として消費）

### キーバインド

| Key            | 画面        | Action                          |
| -------------- | ----------- | ------------------------------- |
| `tab`          | PBI入力     | PBI番号 / リポジトリ / 本文 / 補足 のフィールド切替 |
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
  refinement.yml          メタ情報 + 各セクションへの参照
  pbi.md                  入力したPBI（参照用に保存）
  notes.md                入力した補足メモ（あれば／参照用に保存）
  po_questions.md         POへの確認事項
  implementation/         実装内容（ディレクトリ。実装ステップごとに1ファイル）
    01-design.md
    02-implement.md
    ...
  unit_tests.md           単体テストのテストケース
  integration_tests/      統合テスト（ディレクトリ。E2Eシナリオごとに1ファイル）
    01-happy-path.md
    02-validation-error.md
    ...
```

実装内容は **実装ステップごとのマークダウン** として `implementation/` 配下に出力されます
（`01-`, `02-` のゼロ埋め連番で順序を保持）。`refinement.yml` の `implementation` セクションは
各ステップを `steps` のリストとして参照します:

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
    - id: implementation
      title: 実装内容
      steps:
        - title: 設計方針を決める
          body_file: implementation/01-design.md
        - title: コア実装
          body_file: implementation/02-implement.md
    - ...
```

> `~/.schritt` の場所は `SCHRITT_HOME` 環境変数で上書きできます（主にテスト用）。

### AI の呼び出しについて（plugin / skill）

AIへの指示は **`plugin/` という1つのプラグイン**にまとめてあり、4つの skill
（`refine-pbi` / `implement-step` / `implement-plan` / `verify-e2e`）を `plugin/skills/<name>/SKILL.md`
として同梱しています。各ランタイムにインストールし、**名前で起動**します。

| ランタイム | インストール先 | 起動構文 | 備考 |
| ---------- | -------------- | -------- | ---- |
| Claude Code | `~/.claude/skills/schritt`（skills-dir plugin） | `/schritt:refine-pbi <dir>` | `schritt@skills-dir` として自動ロード・名前空間付き |
| OpenAI Codex CLI | `~/.agents/skills/<name>`（A）or プラグイン（B） | `$refine-pbi <dir>` | A=per-skill symlink / B=marketplace 経由 |

各段階は次の流れで動きます（`internal/refine` / `implement` / `verify`）。

1. 入力（PBI / 実装ステップ / E2Eシナリオ）を一時作業ディレクトリ `<dir>` に書き出す
2. 上表の構文で **skill を名前で起動**（`<dir>` ＋ `--repo` / `--url` 等を渡す）
3. skill が `<dir>` に書いた成果物（markdown / レポート / スクショ）を読み戻し、`~/.schritt/...`
   に整形・保存する

skill 本体はプラグイン1箇所なので、指示を変えたいときは該当 `SKILL.md` を編集するだけです。
エンジンごとの起動引数の差は `internal/agent` に集約しています。

#### プラグイン / skill のインストール

```sh
# Claude Code 用: plugin/ を ~/.claude/skills/schritt にシンボリックリンク
# （skills-dir plugin として自動ロード。claude 再起動後 `claude plugin list` で確認）
make install-plugin
make uninstall-plugin

# Codex CLI 用（2通り。どちらか一方）:
# (A) skill として symlink（推奨・編集が即反映・$refine-pbi で起動）
make install-codex-skills      # = scripts/install-codex.sh
make uninstall-codex-skills

# (B) プラグインとして導入（ローカルmarketplace経由・@/$で起動）
make install-codex-plugin      # = codex plugin marketplace add <repo>
make uninstall-codex-plugin
```

> **Codex のインストール2方式の違い**
> - (A) `~/.agents/skills/<name>` へ **ディレクトリ symlink**。Codex はファイル単位の symlink を
>   落とす ([openai/codex#15756]) ためディレクトリごとリンクします。編集が即反映され、確実。
> - (B) `.agents/plugins/marketplace.json`（`./plugin` を参照）を `codex plugin marketplace add` で
>   登録し、**プラグインとして**導入。ただし Codex はプラグインを `~/.codex/plugins/cache/` へ
>   **コピー**して読み込むため（symlink ではない）、**skill を編集したら再実行が必要**です。
>   `codex plugin` サブコマンド名は codex のバージョンに依存します。
>
> どちらの方式でも、schritt からの起動名は `$refine-pbi` のまま動きます（プラグイン同梱 skill も
> skill 名で起動可能。`@schritt` は対話時にプラグインを明示選択するための構文）。

#### チャットインターフェースから使う

インストール後は **どちらのチャットからも** 同じ skill を使えます。

- Claude Code: PBIを貼り付けて「リファインメントして」と頼むか、`/schritt:refine-pbi` を起動
- Codex CLI: `$refine-pbi` を起動

skill の `description` を見て自動的に選択されます。`<dir>` を渡さずチャットで直接
PBIを貼った場合は、各セクションを返信としてそのまま出力します（SKILL.md に明記）。

AI を使わずに UI を試したい場合は `--demo` を使ってください（プラグイン不要）。

## 実装

リファインメント済みの実装計画を、**実装ステップごとに実装**するステージです。各ステップで
`implement-step` skill を起動し、対象リポジトリにコードと単体テストを実装したうえで、
**何を実装し・どんな単体テストを書いたか** のレポートをステップごとに出力します。

```sh
schritt implement --pbi 42                       # PBI #42 の最新リファインメントを実装
schritt implement --pbi 42 --engine codex        # codex で実行
schritt implement <refinement-dir>               # ディレクトリ指定
schritt implement --pbi 42 --repo ~/proj         # 対象リポジトリを上書き（複数可）
schritt implement --pbi 42 --step 2              # 2番目のステップだけ実装
schritt implement --pbi 42 --demo                # AIなし（サンプルレポート）
```

- 対象リポジトリは `--repo`（複数可）。省略時は refinement.yml の `repo_paths` を使います。
- 各ステップのレポートは `~/.schritt/pbi-{番号}/{日時}/reports/` に、対応する実装ステップと
  同じファイル名（`01-design.md` 等）で保存されます。
- 実行中は AI（claude / codex）の出力をそのまま端末にストリーム表示します。

```
~/.schritt/pbi-42/{日時}/
  implementation/01-design.md     ← 実装計画（ステップ）
  ...
  reports/01-design.md            ← 実装レポート（実装内容 + 書いた単体テスト）
  reports/02-implement.md
  ...
```

> `implement-step` skill も同じ `plugin/` に同梱され、`make install-plugin`（claude）/
> `scripts/install-codex.sh`（codex）でまとめてインストールされます。
> 起動の仕組み（名前で起動・両ランタイム共有）は `internal/agent` に共通化しています。

### チャットから一括実装（implement-plan skill）

`schritt implement` 相当を **チャット（Claude Code / Codex）から**行う `implement-plan` skill も
同梱しています。リファインメント結果ディレクトリを渡すと、実装ステップを**先頭から順に一括実装**し、
各ステップのレポートを `reports/` に出力します（`schritt implement` と同じレイアウト＝`schritt view`
で閲覧可）。

```
# Claude Code
/schritt:implement-plan ~/.schritt/pbi-42/20260620-103000 --repo ~/proj

# Codex CLI
$implement-plan ~/.schritt/pbi-42/20260620-103000 --repo ~/proj
```

1ステップだけ対話的に実装したいときは `implement-step` を使います（前述）。

## 動作確認（verify）

リファインメントの **E2Eシナリオ**（統合テスト）に沿って、**CDP経由でChromeを操作**して
動作確認するステージです。シナリオごとに `verify-e2e` skill を起動し、ブラウザMCP
（[chrome-devtools-mcp]）でページ操作・アサート・スクリーンショット取得を行い、合否レポートを
出力します。

```sh
schritt verify --pbi 42 --url http://localhost:3000    # 全シナリオを検証
schritt verify --pbi 42 --url ... --engine codex       # codex で実行
schritt verify --pbi 42 --url ... --step 1             # 1番目のシナリオだけ
schritt verify --pbi 42 --demo                         # ブラウザ/AIなし（サンプル）
```

- 検証対象アプリは **`--url` で指定**（起動済みであること）。
- シナリオごとに `verification/<シナリオ名>.md`（PASS/FAIL＋観察）と、
  `verification/screenshots/<シナリオ名>/` にスクリーンショットを保存します。

```
~/.schritt/pbi-42/{日時}/
  integration_tests/01-happy-path.md     ← E2Eシナリオ（refinement由来）
  ...
  verification/01-happy-path.md          ← 検証レポート（合否＋観察）
  verification/screenshots/01-happy-path/01-top.png ...
```

> **前提**: `chrome-devtools-mcp`（`npx -y chrome-devtools-mcp@latest`）と Chrome が必要です。
> ブラウザMCPは `internal/agent` 経由で claude には `--mcp-config`、codex には `-c mcp_servers...`
> ＋ネットワーク開放（`-c sandbox_workspace_write.network_access=true`）として渡します。
> Playwright MCP 等への差し替えも可能です（`verify.DefaultBrowserMCP` を変えるだけ）。

## 結果・レポートをTUIで閲覧（view）

`schritt view` で、既存のリファインメント結果と各レポートを TUI で閲覧できます。
リファインメント内容に加え、**実装レポート**（`reports/`）・**検証レポート**（`verification/`）が
あれば一覧に並びます。

```sh
schritt view --pbi 42            # PBI #42 の最新を開く
schritt view <refinement-dir>    # ディレクトリ指定
```

一覧例（`j/k` 移動・`Enter` で本文表示・`l`/`Esc` で戻る）:

```
POへの確認事項
実装内容 ▸ 1. 設計方針を決める
...
統合テスト（E2Eシナリオ） ▸ 1. 正常系
...
実装レポート ▸ 1. 設計方針を決める      ← reports/ があれば
...
検証レポート ▸ 1. 正常系                ← verification/ があれば
```

## 開発

```sh
make run ARGS="refinement --demo"   # build + 起動
make build                          # bin/schritt をビルド
make test                           # テスト
make lint                           # golangci-lint
```

## レイアウト

```
cmd/schritt/main.go       Cobra エントリポイント。refinement / implement / verify サブコマンド。
cmd/schritt/implement.go  implement サブコマンド（ステップごとに実装＋レポート保存）。
cmd/schritt/verify.go     verify サブコマンド（シナリオごとにブラウザ検証＋レポート保存）。
cmd/schritt/view.go       view サブコマンド（結果・各レポートをTUIで閲覧）。
internal/model/           ドメイン型（Refinement / Section / Step / PBIMeta）。
internal/agent/           AI CLI 起動の共通基盤。skill を名前で起動（claude/codex）＋MCP/ネットワーク。
internal/refine/          リファインメント段階。Refiner と Claude/Codex/Demo 実装。
internal/implement/       実装段階。Implementer と Claude/Codex/Demo 実装。
internal/verify/          動作確認段階。Verifier と Claude/Codex/Demo 実装（ブラウザMCP）。
internal/store/           refinement.yml + markdown + reports/ + verification/ の保存・読み込み。
internal/tui/             結果・レポートを見る Bubble Tea TUI（refinement 実行 / view 閲覧）。
plugin/.claude-plugin/    Claude Code プラグインのマニフェスト (plugin.json)。
plugin/.codex-plugin/     Codex プラグインのマニフェスト (plugin.json)。
plugin/skills/<name>/     refine-pbi / implement-step / implement-plan / verify-e2e の各 SKILL.md（プラグイン同梱）。
.agents/plugins/          Codex ローカルマーケットプレース (marketplace.json)。プラグイン導入用。
scripts/install-codex.sh  Codex CLI 用 skill インストーラ (~/.agents/skills へ symlink)。
```

`refine.Refiner` / `implement.Implementer` / `verify.Verifier` を差し替え口にしているため、
AI 呼び出し方法を変えても上位層は影響を受けません。テストや `--demo` は Demo 実装を使います。
エンジン固有の起動引数（サンドボックス・ディレクトリ付与・MCP・ネットワーク・`/name` vs `$name`）は
`internal/agent` に集約してあり、refine / implement / verify の各段階で共有します。

## 今後のステップ（予定）

`pre-exec review`（実装計画レビュー）・PR作成のサブコマンドを、同じ `cmd` / `internal/agent` /
`internal/store` の構造に沿って追加していく想定です。

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Claude Code]: https://docs.claude.com/en/docs/claude-code
[Codex CLI]: https://github.com/openai/codex
[Codex skills]: https://developers.openai.com/codex/skills
[Claude Code skill]: https://docs.claude.com/en/docs/claude-code/skills
[openai/codex#15756]: https://github.com/openai/codex/issues/15756
[chrome-devtools-mcp]: https://github.com/ChromeDevTools/chrome-devtools-mcp
