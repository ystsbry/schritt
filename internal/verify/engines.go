package verify

import (
	"context"
	"fmt"
	"io"

	"github.com/ystsbry/schritt/internal/agent"
)

// ClaudeVerifier verifies a scenario via the `claude` CLI, invoking the
// verify-e2e skill (`/verify-e2e <dir> --url ...`) with a browser MCP.
type ClaudeVerifier struct {
	Bin    string          // override binary; "" → "claude"
	Model  string          // optional --model
	MCP    agent.MCPServer // browser MCP; zero value → DefaultBrowserMCP()
	Stream io.Writer       // optional: receives the agent's live output
}

func (c *ClaudeVerifier) Verify(ctx context.Context, in Input) (Result, error) {
	return runScenario(ctx, in, agent.Claude, c.Bin, c.Model, mcpOrDefault(c.MCP), c.Stream)
}

// CodexVerifier verifies a scenario via the `codex` CLI, invoking the
// verify-e2e skill (`$verify-e2e <dir> --url ...`) with a browser MCP and
// network egress enabled.
type CodexVerifier struct {
	Bin    string
	Model  string
	MCP    agent.MCPServer
	Stream io.Writer
}

func (c *CodexVerifier) Verify(ctx context.Context, in Input) (Result, error) {
	return runScenario(ctx, in, agent.Codex, c.Bin, c.Model, mcpOrDefault(c.MCP), c.Stream)
}

func mcpOrDefault(m agent.MCPServer) agent.MCPServer {
	if m.Name == "" || m.Command == "" {
		return DefaultBrowserMCP()
	}
	return m
}

// DemoVerifier returns a canned PASS report without a browser, AI, or running
// app. It exists so the verify flow can be exercised end-to-end (via `schritt
// verify --demo`) and in tests.
type DemoVerifier struct{}

func (DemoVerifier) Verify(_ context.Context, in Input) (Result, error) {
	title := in.ScenarioTitle
	if title == "" {
		title = "E2Eシナリオ"
	}
	return Result{Report: fmt.Sprintf(`# %s

## 判定
- PASS (demo)

## 確認した操作と結果
- (demo) シナリオの操作手順に沿って動作確認したと仮定したレポートです。
- 実際のブラウザ操作・スクリーンショット取得は行っていません（--demo モード）。

## スクリーンショット
- (demo) なし

## 補足
- 特になし
`, title)}, nil
}
