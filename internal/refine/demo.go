package refine

import (
	"context"
	"fmt"
)

// DemoRefiner returns canned sections without calling any AI. It exists so the
// TUI flow (paste PBI → refine → view) can be exercised end-to-end — in tests
// and via `schritt refinement --demo` — with no `claude` dependency.
type DemoRefiner struct{}

func (DemoRefiner) Refine(_ context.Context, in Input) (Result, error) {
	n := in.PBINumber
	return Result{
		POQuestions: fmt.Sprintf(`# POへの確認事項 (PBI #%d)

- 受け入れ条件の「対応済み」の定義を確認したい（UI表示まで？API応答まで？）。
- 想定外の入力（空・極端に長い値）時の挙動は仕様化が必要か。
- 既存機能との優先順位・リリース時期の制約はあるか。
`, n),
		Implementation: []Doc{
			{
				File:  "01-design.md",
				Title: "設計方針を決める",
				Body: `# 設計方針を決める

- 入力 → 変換 → 出力の3層に分け、変換層に本PBIのロジックを集約する。
- 外部I/Oは interface 越しにし、テスト時にスタブ可能にする。
`,
			},
			{
				File:  "02-implement.md",
				Title: "コア実装",
				Body: `# コア実装

- ` + "`internal/...`" + ` に新規パッケージを追加し、変換層のロジックを実装する。
- エントリポイントは既存のコマンドから呼び出す。
`,
			},
			{
				File:  "03-wire-up.md",
				Title: "結線と仕上げ",
				Body: `# 結線と仕上げ

- 既存コマンドから新パッケージを呼び出すよう結線する。
- エラーハンドリングとログを整える。
`,
			},
		},
		UnitTests: `# 単体テストのテストケース

| 観点 | 入力 | 期待結果 |
| ---- | ---- | -------- |
| 正常系 | 代表的な入力 | 期待する出力 |
| 境界値 | 空文字 / 最大長 | エラーにならず規定の挙動 |
| 異常系 | 不正な値 | 明示的なエラーを返す |
`,
		Integration: []Doc{
			{
				File:  "01-happy-path.md",
				Title: "正常系: 一連の操作が完了する",
				Body: `# 正常系: 一連の操作が完了する

## 前提
- アプリが起動済みで、トップページにアクセスできる。

## 操作手順
1. トップページを開く。
2. 主要な操作（例: フォーム入力 → 送信）を行う。

## 期待結果
- 成功時のメッセージ/画面が表示される。
- エラー表示が出ない。
`,
			},
			{
				File:  "02-validation-error.md",
				Title: "異常系: 入力エラーが表示される",
				Body: `# 異常系: 入力エラーが表示される

## 前提
- アプリが起動済み。

## 操作手順
1. フォームに不正な値（空・極端に長い値）を入力して送信する。

## 期待結果
- バリデーションエラーが画面に表示される。
- データは送信/保存されない。
`,
			},
		},
	}, nil
}
