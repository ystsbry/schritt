package agent

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestClaudeArgsInvokeSkillByName(t *testing.T) {
	s := Spec{Engine: Claude, Model: "opus", SkillName: "refine-pbi", WorkDir: "/work"}
	args := s.Args()
	joined := strings.Join(args, " ")
	for _, want := range []string{"--model opus", "--add-dir /work", "--permission-mode acceptEdits", "--print"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("claude Args missing %q in %v", want, args)
		}
	}
	if last := args[len(args)-1]; last != "/refine-pbi /work" {
		t.Fatalf("expected trailing skill invocation, got %q", last)
	}
}

func TestCodexArgsInvokeSkillByName(t *testing.T) {
	s := Spec{Engine: Codex, Model: "gpt-5-codex", SkillName: "implement-step", WorkDir: "/work"}
	args := s.Args()
	if args[0] != "exec" {
		t.Fatalf("expected codex exec first, got %v", args)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--cd /work", "--skip-git-repo-check", "--sandbox workspace-write", "--model gpt-5-codex"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("codex Args missing %q in %v", want, args)
		}
	}
	if last := args[len(args)-1]; last != "$implement-step /work" {
		t.Fatalf("expected trailing skill invocation, got %q", last)
	}
}

func TestArgsGrantExtraDirsAndSkillArgs(t *testing.T) {
	repos := []string{"/repo/front", "/repo/back"}
	skillArgs := []string{"--repo", "/repo/front", "--repo", "/repo/back"}

	c := Spec{Engine: Claude, SkillName: "refine-pbi", WorkDir: "/work", ExtraDirs: repos, SkillArgs: skillArgs}.Args()
	if !strings.Contains(strings.Join(c, " "), "--add-dir /work /repo/front /repo/back") {
		t.Fatalf("claude should grant all dirs via --add-dir: %v", c)
	}
	if last := c[len(c)-1]; last != "/refine-pbi /work --repo /repo/front --repo /repo/back" {
		t.Fatalf("claude invocation should carry skill args, got %q", last)
	}

	x := Spec{Engine: Codex, SkillName: "refine-pbi", WorkDir: "/work", ExtraDirs: repos, SkillArgs: skillArgs}.Args()
	xj := strings.Join(x, " ")
	if !strings.Contains(xj, "--add-dir /repo/front") || !strings.Contains(xj, "--add-dir /repo/back") {
		t.Fatalf("codex should grant each dir via --add-dir: %v", x)
	}
	if last := x[len(x)-1]; last != "$refine-pbi /work --repo /repo/front --repo /repo/back" {
		t.Fatalf("codex invocation should carry skill args, got %q", last)
	}
}

func TestArgsOmitModelWhenEmpty(t *testing.T) {
	if strings.Contains(strings.Join(Spec{Engine: Claude, SkillName: "x", WorkDir: "/w"}.Args(), " "), "--model") {
		t.Fatalf("claude should omit --model when empty")
	}
	if strings.Contains(strings.Join(Spec{Engine: Codex, SkillName: "x", WorkDir: "/w"}.Args(), " "), "--model") {
		t.Fatalf("codex should omit --model when empty")
	}
}

func TestClaudeArgsInjectMCPAndAllowedTools(t *testing.T) {
	s := Spec{
		Engine: Claude, SkillName: "verify-e2e", WorkDir: "/work",
		MCPServers:   []MCPServer{{Name: "chrome-devtools", Command: "npx", Args: []string{"-y", "chrome-devtools-mcp@latest"}}},
		AllowedTools: []string{"mcp__chrome-devtools"},
	}
	joined := strings.Join(s.Args(), " ")
	if !strings.Contains(joined, "--mcp-config") {
		t.Fatalf("claude Args should include --mcp-config: %s", joined)
	}
	if !strings.Contains(joined, `"chrome-devtools"`) || !strings.Contains(joined, "chrome-devtools-mcp@latest") {
		t.Fatalf("mcp-config JSON missing server: %s", joined)
	}
	if !strings.Contains(joined, "--allowedTools mcp__chrome-devtools") {
		t.Fatalf("claude Args should allow the MCP tools: %s", joined)
	}
}

func TestCodexArgsInjectNetworkAndMCP(t *testing.T) {
	s := Spec{
		Engine: Codex, SkillName: "verify-e2e", WorkDir: "/work",
		NetworkAccess: true,
		MCPServers:    []MCPServer{{Name: "chrome-devtools", Command: "npx", Args: []string{"-y", "chrome-devtools-mcp@latest"}}},
	}
	joined := strings.Join(s.Args(), " ")
	if !strings.Contains(joined, "sandbox_workspace_write.network_access=true") {
		t.Fatalf("codex Args should open network egress: %s", joined)
	}
	if !strings.Contains(joined, "mcp_servers.chrome-devtools.command=") {
		t.Fatalf("codex Args should register the MCP server: %s", joined)
	}
}

func TestRunRejectsUnknownEngine(t *testing.T) {
	if err := Run(context.Background(), Spec{Engine: "bogus", SkillName: "x", WorkDir: "/w"}); err == nil {
		t.Fatalf("expected error for unknown engine")
	}
}

func TestRunReportsMissingBinary(t *testing.T) {
	err := Run(context.Background(), Spec{Engine: Codex, Bin: "schritt-no-such-bin", SkillName: "x", WorkDir: "/w"})
	if err == nil {
		t.Fatalf("expected ErrCLINotFound for missing binary")
	}
}

// TestRunExecutesBinary runs a fake CLI that writes a marker file into its
// work dir, proving Args/Run wire up cwd/access correctly for both engines.
func TestRunExecutesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only fake CLI")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake")
	script := `#!/usr/bin/env bash
for a in "$@"; do if [ -d "$a" ]; then echo ok > "$a/marker"; fi; done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	work := t.TempDir()
	if err := Run(context.Background(), Spec{Engine: Claude, Bin: bin, SkillName: "x", WorkDir: work}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "marker")); err != nil {
		t.Fatalf("expected fake CLI to write marker: %v", err)
	}
}
