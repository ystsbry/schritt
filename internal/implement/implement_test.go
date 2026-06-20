package implement

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ystsbry/schritt/internal/agent"
)

func TestDemoImplementerProducesReport(t *testing.T) {
	res, err := DemoImplementer{}.Implement(context.Background(), Input{StepTitle: "設計"})
	if err != nil {
		t.Fatalf("Implement: %v", err)
	}
	for _, want := range []string{"# 設計", "## 実装した内容", "## 書いた単体テスト"} {
		if !strings.Contains(res.Report, want) {
			t.Fatalf("report missing %q:\n%s", want, res.Report)
		}
	}
}

func TestRunStepRequiresRepo(t *testing.T) {
	_, err := runStep(context.Background(), Input{StepBody: "x"}, agent.Claude, "claude", "", nil)
	if err == nil || !strings.Contains(err.Error(), "リポジトリ") {
		t.Fatalf("expected repo-required error, got %v", err)
	}
}

func TestRunStepRejectsEmptyStep(t *testing.T) {
	if _, err := runStep(context.Background(), Input{}, agent.Claude, "claude", "", nil); err == nil {
		t.Fatalf("expected error for empty step body")
	}
}

// fakeAgent writes a report.md into the work dir (found as an existing dir in
// args), simulating the implement-step skill. It also requires a repo arg to
// be present so we exercise the repo plumbing.
func fakeAgent(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only fake CLI")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-impl")
	script := `#!/usr/bin/env bash
work=""
for a in "$@"; do
  case "$a" in
    *report-marker*) ;;
  esac
  if [ -d "$a" ] && [ -f "$a/step.md" ]; then work="$a"; fi
done
if [ -z "$work" ]; then echo "no work dir with step.md" >&2; exit 1; fi
printf '# 実装ステップ\n\n## 実装した内容\n- done\n\n## 書いた単体テスト\n- added\n' > "$work/report.md"
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	return bin
}

func TestRunStepEndToEnd(t *testing.T) {
	bin := fakeAgent(t)
	repo := t.TempDir()
	res, err := runStep(context.Background(), Input{
		StepTitle: "設計",
		StepBody:  "# 設計\n\n本文",
		PBIBody:   "# PBI",
		RepoPaths: []string{repo},
	}, agent.Claude, bin, "", nil)
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if !strings.Contains(res.Report, "## 実装した内容") || !strings.Contains(res.Report, "## 書いた単体テスト") {
		t.Fatalf("unexpected report:\n%s", res.Report)
	}
}

func TestRunStepReportsMissingReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "noop")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write noop: %v", err)
	}
	repo := t.TempDir()
	_, err := runStep(context.Background(), Input{StepBody: "x", RepoPaths: []string{repo}}, agent.Codex, bin, "", nil)
	if err == nil || !strings.Contains(err.Error(), "report.md") {
		t.Fatalf("expected missing-report error, got %v", err)
	}
}
