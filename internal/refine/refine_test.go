package refine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The engine-specific argv construction is unit-tested in internal/agent. Here
// we exercise the refine driver end-to-end (work-dir setup → invocation →
// read-back) against a fake CLI.

func TestRefineRejectsBadRepoPath(t *testing.T) {
	bin := fakeCLI(t)
	_, err := (&ClaudeRefiner{Bin: bin}).Refine(context.Background(), Input{
		PBINumber: 1, PBIBody: "x", RepoPaths: []string{"/no/such/dir/schritt-test"},
	}, nil)
	if err == nil {
		t.Fatalf("expected error for non-existent repo path")
	}
}

// fakeCLI writes a small executable that mimics an agent running the skill: it
// scans its args for an existing directory (the work dir, passed via
// --add-dir / --cd) and writes the po_questions/, implementation/ and
// integration_tests/ section directories into it. This exercises the full
// read-back path for both engines without a real CLI/skill.
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
mkdir -p "$target/po_questions"
printf '# 受け入れ条件\n\n本文\n' > "$target/po_questions/01-acceptance.md"
printf '# 想定外入力\n\n本文\n' > "$target/po_questions/02-edge.md"
mkdir -p "$target/implementation"
printf '# 設計\n\n本文\n' > "$target/implementation/01-design.md"
printf '# 実装\n\n本文\n' > "$target/implementation/02-build.md"
mkdir -p "$target/integration_tests"
printf '# 正常系\n\n本文\n' > "$target/integration_tests/01-happy.md"
printf '# 異常系\n\n本文\n' > "$target/integration_tests/02-error.md"
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
		res, err := r.Refine(context.Background(), in, nil)
		if err != nil {
			t.Fatalf("%s Refine: %v", name, err)
		}
		// PO questions are read back as ordered docs from po_questions/.
		if len(res.POQuestions) != 2 {
			t.Fatalf("%s: expected 2 PO questions, got %d", name, len(res.POQuestions))
		}
		if res.POQuestions[0].File != "01-acceptance.md" || res.POQuestions[1].File != "02-edge.md" {
			t.Fatalf("%s: PO questions out of order: %+v", name, res.POQuestions)
		}
		if res.POQuestions[0].Title != "受け入れ条件" {
			t.Fatalf("%s: PO question title should come from heading, got %q", name, res.POQuestions[0].Title)
		}
		// Implementation is read back as ordered docs from implementation/.
		if len(res.Implementation) != 2 {
			t.Fatalf("%s: expected 2 implementation steps, got %d", name, len(res.Implementation))
		}
		if res.Implementation[0].File != "01-design.md" || res.Implementation[1].File != "02-build.md" {
			t.Fatalf("%s: implementation steps out of order: %+v", name, res.Implementation)
		}
		if res.Implementation[0].Title != "設計" {
			t.Fatalf("%s: step title should come from heading, got %q", name, res.Implementation[0].Title)
		}
		// Integration (E2E) scenarios are read back the same way.
		if len(res.Integration) != 2 {
			t.Fatalf("%s: expected 2 integration scenarios, got %d", name, len(res.Integration))
		}
		if res.Integration[0].Title != "正常系" {
			t.Fatalf("%s: scenario title should come from heading, got %q", name, res.Integration[0].Title)
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
	_, err := (&CodexRefiner{Bin: bin}).Refine(context.Background(), Input{PBINumber: 1, PBIBody: "x"}, nil)
	if err == nil {
		t.Fatalf("expected error when no section files are produced")
	}
	if !strings.Contains(err.Error(), "po_questions/*.md") || !strings.Contains(err.Error(), "install-codex") {
		t.Fatalf("expected missing-file + install hint, got: %v", err)
	}
}

func TestRefinerMissingBinary(t *testing.T) {
	_, err := (&CodexRefiner{Bin: "schritt-no-such-codex-bin"}).Refine(context.Background(), Input{PBINumber: 1, PBIBody: "x"}, nil)
	if err == nil {
		t.Fatalf("expected error for missing binary")
	}
}
