package refine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestClaudeArgsInvokesSkillByName(t *testing.T) {
	args := claudeArgs("opus", "/work", nil)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--model opus", "--add-dir /work", "--permission-mode acceptEdits", "--print"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("claudeArgs missing %q in %v", want, args)
		}
	}
	// The skill is invoked by name as the --print prompt (trailing positional,
	// so --add-dir can't swallow it).
	if last := args[len(args)-1]; last != "/refine-pbi /work" {
		t.Fatalf("expected last arg to invoke skill by name, got %q", last)
	}
}

func TestCodexArgsInvokesSkillByName(t *testing.T) {
	args := codexArgs("gpt-5-codex", "/work", nil)
	if args[0] != "exec" {
		t.Fatalf("expected codex exec subcommand first, got %v", args)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--cd /work", "--skip-git-repo-check", "--sandbox workspace-write", "--model gpt-5-codex"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("codexArgs missing %q in %v", want, args)
		}
	}
	// Codex invokes the skill via its "$name" syntax as the trailing positional.
	if last := args[len(args)-1]; last != "$refine-pbi /work" {
		t.Fatalf("expected last arg to invoke skill by name, got %q", last)
	}
}

func TestArgsIncludeMultipleReposWhenGiven(t *testing.T) {
	repos := []string{"/repo/front", "/repo/back"}
	// claude: every repo is granted via the variadic --add-dir and passed to
	// the skill as a repeated --repo flag.
	ca := claudeArgs("", "/work", repos)
	cj := strings.Join(ca, " ")
	if !strings.Contains(cj, "--add-dir /work /repo/front /repo/back") {
		t.Fatalf("claudeArgs should grant all repos via --add-dir: %v", ca)
	}
	if last := ca[len(ca)-1]; last != "/refine-pbi /work --repo /repo/front --repo /repo/back" {
		t.Fatalf("claude skill invocation should pass each --repo, got %q", last)
	}
	// codex: each repo via its own --add-dir; repeated --repo in the invocation.
	xa := codexArgs("", "/work", repos)
	xj := strings.Join(xa, " ")
	if !strings.Contains(xj, "--add-dir /repo/front") || !strings.Contains(xj, "--add-dir /repo/back") {
		t.Fatalf("codexArgs should grant each repo via --add-dir: %v", xa)
	}
	if last := xa[len(xa)-1]; last != "$refine-pbi /work --repo /repo/front --repo /repo/back" {
		t.Fatalf("codex skill invocation should pass each --repo, got %q", last)
	}
}

func TestArgsOmitModelWhenEmpty(t *testing.T) {
	if strings.Contains(strings.Join(claudeArgs("", "/w", nil), " "), "--model") {
		t.Fatalf("claudeArgs should omit --model when empty")
	}
	if strings.Contains(strings.Join(codexArgs("", "/w", nil), " "), "--model") {
		t.Fatalf("codexArgs should omit --model when empty")
	}
}

func TestRefineRejectsBadRepoPath(t *testing.T) {
	bin := fakeCLI(t)
	_, err := (&ClaudeRefiner{Bin: bin}).Refine(context.Background(), Input{
		PBINumber: 1, PBIBody: "x", RepoPaths: []string{"/no/such/dir/schritt-test"},
	})
	if err == nil {
		t.Fatalf("expected error for non-existent repo path")
	}
}

// fakeCLI writes a small executable that mimics an agent running the skill: it
// scans its args for an existing directory (the work dir, passed via
// --add-dir / --cd) and writes the four section files into it. This exercises
// the full runCLI read-back path for both engines without a real CLI/skill.
func fakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI shell script is POSIX-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-agent")
	script := `#!/usr/bin/env bash
target=""
for a in "$@"; do
  if [ -d "$a" ]; then target="$a"; fi
done
if [ -z "$target" ]; then echo "no work dir in args" >&2; exit 1; fi
for f in po_questions implementation unit_tests integration_tests; do
  printf '# %s\n\n本文\n' "$f" > "$target/$f.md"
done
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake CLI: %v", err)
	}
	return path
}

func TestRefinersRunEndToEnd(t *testing.T) {
	bin := fakeCLI(t)
	in := Input{PBINumber: 9, PBIBody: "# PBI\n\n本文", Notes: "会議メモ"}

	refiners := map[string]Refiner{
		"claude": &ClaudeRefiner{Bin: bin},
		"codex":  &CodexRefiner{Bin: bin},
	}
	for name, r := range refiners {
		res, err := r.Refine(context.Background(), in)
		if err != nil {
			t.Fatalf("%s Refine: %v", name, err)
		}
		for field, body := range map[string]string{
			"POQuestions":      res.POQuestions,
			"Implementation":   res.Implementation,
			"UnitTests":        res.UnitTests,
			"IntegrationTests": res.IntegrationTests,
		} {
			if strings.TrimSpace(body) == "" {
				t.Fatalf("%s: %s is empty", name, field)
			}
		}
	}
}

// TestRefinerReportsMissingSections covers the "skill not installed" path: a
// fake CLI that succeeds but writes nothing should yield an error mentioning
// the missing files and an install hint.
func TestRefinerReportsMissingSections(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "noop")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write noop: %v", err)
	}
	_, err := (&CodexRefiner{Bin: bin}).Refine(context.Background(), Input{PBINumber: 1, PBIBody: "x"})
	if err == nil {
		t.Fatalf("expected error when no section files are produced")
	}
	if !strings.Contains(err.Error(), "po_questions.md") || !strings.Contains(err.Error(), "install-codex") {
		t.Fatalf("expected missing-file + install hint, got: %v", err)
	}
}

func TestRefinerMissingBinary(t *testing.T) {
	_, err := (&CodexRefiner{Bin: "schritt-no-such-codex-bin"}).Refine(context.Background(), Input{PBINumber: 1, PBIBody: "x"})
	if err == nil {
		t.Fatalf("expected error for missing binary")
	}
}
