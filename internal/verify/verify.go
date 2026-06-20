// Package verify is the verification pipeline stage: driving Chrome via CDP to
// confirm the implemented feature against the refinement's E2E scenarios. For
// each scenario it runs the verify-e2e skill (via the agent package), which
// operates a real browser through a CDP/browser MCP and writes a per-scenario
// report plus screenshots.
package verify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ystsbry/schritt/internal/agent"
)

// skillName is the verify-e2e skill, invoked by name from each runtime.
const skillName = "verify-e2e"

// DefaultBrowserMCP is the CDP/browser MCP server used to drive Chrome. It is
// overridable so users can swap in Playwright MCP, pin a version, etc.
func DefaultBrowserMCP() agent.MCPServer {
	return agent.MCPServer{
		Name:    "chrome-devtools",
		Command: "npx",
		Args:    []string{"-y", "chrome-devtools-mcp@latest"},
	}
}

// Input is one E2E scenario to verify.
type Input struct {
	// ScenarioTitle is the scenario's human label (for messages).
	ScenarioTitle string
	// ScenarioBody is the scenario markdown (前提/操作手順/期待結果), written to
	// scenario.md for the skill.
	ScenarioBody string
	// PBIBody is the original PBI markdown for context (optional).
	PBIBody string
	// AppURL is the running application's URL. Required for a real run.
	AppURL string
	// RepoPaths are repositories the skill may consult (optional).
	RepoPaths []string
}

// Screenshot is one captured image, read back from the run.
type Screenshot struct {
	Name string // base filename, e.g. "01-top.png"
	Data []byte
}

// Result holds the per-scenario verification output.
type Result struct {
	Report      string       // markdown report (PASS/FAIL + observations)
	Screenshots []Screenshot // captured evidence
}

// Verifier verifies one E2E scenario.
type Verifier interface {
	Verify(ctx context.Context, in Input) (Result, error)
}

const reportFile = "report.md"

// runScenario is the engine-agnostic driver: write the scenario into a work
// dir, invoke verify-e2e by name with a browser MCP available, and read back
// the report and screenshots. stream, if set, receives the agent's live output.
func runScenario(ctx context.Context, in Input, engine, bin, model string, mcp agent.MCPServer, stream io.Writer) (Result, error) {
	if strings.TrimSpace(in.ScenarioBody) == "" {
		return Result{}, errors.New("ScenarioBody is empty")
	}
	if strings.TrimSpace(in.AppURL) == "" {
		return Result{}, errors.New("AppURL is required (--url)")
	}

	var repoPaths []string
	for _, raw := range in.RepoPaths {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			return Result{}, fmt.Errorf("resolve repo path %q: %w", raw, err)
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return Result{}, fmt.Errorf("repo path %q is not a directory", raw)
		}
		repoPaths = append(repoPaths, abs)
	}

	work, err := os.MkdirTemp("", "schritt-verify-*")
	if err != nil {
		return Result{}, fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()

	if err := os.WriteFile(filepath.Join(work, "scenario.md"), []byte(in.ScenarioBody), 0o644); err != nil {
		return Result{}, fmt.Errorf("write scenario.md: %w", err)
	}
	if b := strings.TrimSpace(in.PBIBody); b != "" {
		if err := os.WriteFile(filepath.Join(work, "pbi.md"), []byte(in.PBIBody), 0o644); err != nil {
			return Result{}, fmt.Errorf("write pbi.md: %w", err)
		}
	}

	skillArgs := []string{"--url", in.AppURL}
	for _, r := range repoPaths {
		skillArgs = append(skillArgs, "--repo", r)
	}
	err = agent.Run(ctx, agent.Spec{
		Engine:        engine,
		Bin:           bin,
		Model:         model,
		SkillName:     skillName,
		WorkDir:       work,
		ExtraDirs:     repoPaths,
		SkillArgs:     skillArgs,
		MCPServers:    []agent.MCPServer{mcp},
		AllowedTools:  []string{"mcp__" + mcp.Name},
		NetworkAccess: true, // codex: reach the app URL + launch Chrome
		Stdout:        stream,
		Stderr:        stream,
	})
	if err != nil {
		return Result{}, err
	}

	body, err := os.ReadFile(filepath.Join(work, reportFile))
	if err != nil {
		return Result{}, fmt.Errorf("%s が %s を生成しませんでした。skill 未インストール、または検証が完了しなかった可能性があります。\n%s",
			engine, reportFile, installHint(engine))
	}
	shots, err := readScreenshots(filepath.Join(work, "screenshots"))
	if err != nil {
		return Result{}, err
	}
	return Result{
		Report:      strings.TrimRight(string(body), "\n") + "\n",
		Screenshots: shots,
	}, nil
}

// readScreenshots loads every image file under dir into memory (before the work
// dir is cleaned up). Returns nil when the directory is absent.
func readScreenshots(dir string) ([]Screenshot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read screenshots: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".png", ".jpg", ".jpeg", ".webp":
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var shots []Screenshot
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read screenshot %s: %w", name, err)
		}
		shots = append(shots, Screenshot{Name: name, Data: data})
	}
	return shots, nil
}

// installHint returns the skill-install guidance for the given engine.
func installHint(engine string) string {
	if engine == agent.Codex {
		return `verify-e2e skill が Codex CLI に見つからない可能性があります。
リポジトリのルートで scripts/install-codex.sh を実行し、codex を再起動してください。
また、ブラウザMCP（chrome-devtools-mcp）と Chrome が利用できる必要があります。`
	}
	return `verify-e2e skill が Claude Code に見つからない可能性があります。
リポジトリのルートで make install-plugin を実行してください。
また、ブラウザMCP（chrome-devtools-mcp）と Chrome が利用できる必要があります。`
}
