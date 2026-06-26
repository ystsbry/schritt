package refine

import (
	"context"
	"fmt"
	"time"
)

// DemoRefiner returns canned sections without calling any AI. It exists so the
// TUI flow (paste PBI → refine → view) can be exercised end-to-end — in tests
// and via `schritt refinement --demo` — with no `claude` dependency.
type DemoRefiner struct{}

func (DemoRefiner) Refine(ctx context.Context, in Input, progress func(string)) (Result, error) {
	n := in.PBINumber
	// Emit a few simulated progress lines so `--demo` exercises the live
	// progress view the real engines drive via stream-json.
	if progress != nil {
		for _, line := range []string{
			"claude セッション開始",
			"Read: pbi.md",
			fmt.Sprintf("Skill: refine-pbi (PBI #%d)", n),
			"Write: po_questions/01-acceptance-criteria.md",
			"Write: implementation/01-design.md",
		} {
			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case <-time.After(120 * time.Millisecond):
			}
			progress(line)
		}
	}
	return Result{
		POQuestions: []Doc{
			{
				File:  "01-acceptance-criteria.md",
				Title: "受け入れ条件の定義",
				Body: fmt.Sprintf(`# 受け入れ条件の定義

- PBI #%d の受け入れ条件「対応済み」の定義を確認したい（UI表示まで？API応答まで？）。
`, n),
			},
			{
				File:  "02-edge-cases.md",
				Title: "想定外入力の扱い",
				Body: `# 想定外入力の扱い

- 想定外の入力（空・極端に長い値）時の挙動は仕様化が必要か。
`,
			},
			{
				File:  "03-priority.md",
				Title: "優先順位・リリース制約",
				Body: `# 優先順位・リリース制約

- 既存機能との優先順位・リリース時期の制約はあるか。
`,
			},
		},
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
