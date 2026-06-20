package implement

import (
	"context"
	"fmt"
	"io"

	"github.com/ystsbry/schritt/internal/agent"
)

// ClaudeImplementer implements a step via the `claude` CLI, invoking the
// implement-step skill (`/implement-step <dir>`).
type ClaudeImplementer struct {
	Bin    string    // override binary; "" → "claude"
	Model  string    // optional --model
	Stream io.Writer // optional: receives the agent's live output
}

func (c *ClaudeImplementer) Implement(ctx context.Context, in Input) (Result, error) {
	return runStep(ctx, in, agent.Claude, c.Bin, c.Model, c.Stream)
}

// CodexImplementer implements a step via the `codex` CLI, invoking the
// implement-step skill (`$implement-step <dir>`).
type CodexImplementer struct {
	Bin    string    // override binary; "" → "codex"
	Model  string    // optional --model
	Stream io.Writer // optional: receives the agent's live output
}

func (c *CodexImplementer) Implement(ctx context.Context, in Input) (Result, error) {
	return runStep(ctx, in, agent.Codex, c.Bin, c.Model, c.Stream)
}

// DemoImplementer returns a canned report without calling any AI or touching a
// repository. It exists so the implement flow can be exercised end-to-end
// (via `schritt implement --demo`) and in tests.
type DemoImplementer struct{}

func (DemoImplementer) Implement(_ context.Context, in Input) (Result, error) {
	title := in.StepTitle
	if title == "" {
		title = "実装ステップ"
	}
	return Result{Report: fmt.Sprintf(`# %s

## 実装した内容
- (demo) このステップの計画に沿ってコードを実装したと仮定したレポートです。
- 実際のコード変更は行っていません（--demo モード）。

## 書いた単体テスト
- (demo) 代表的な正常系・境界値・異常系のテストケースを追加した想定です。

## 補足
- 特になし
`, title)}, nil
}
