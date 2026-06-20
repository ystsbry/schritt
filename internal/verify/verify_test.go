package verify

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ystsbry/schritt/internal/agent"
)

func TestDemoVerifierProducesReport(t *testing.T) {
	res, err := DemoVerifier{}.Verify(context.Background(), Input{ScenarioTitle: "ログイン"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, want := range []string{"# ログイン", "## 判定", "PASS"} {
		if !strings.Contains(res.Report, want) {
			t.Fatalf("report missing %q:\n%s", want, res.Report)
		}
	}
}

func TestRunScenarioRequiresURL(t *testing.T) {
	_, err := runScenario(context.Background(), Input{ScenarioBody: "x"}, agent.Claude, "claude", "", DefaultBrowserMCP(), nil)
	if err == nil || !strings.Contains(err.Error(), "url") && !strings.Contains(err.Error(), "URL") {
		t.Fatalf("expected URL-required error, got %v", err)
	}
}

func TestRunScenarioRejectsEmptyScenario(t *testing.T) {
	_, err := runScenario(context.Background(), Input{AppURL: "http://x"}, agent.Claude, "claude", "", DefaultBrowserMCP(), nil)
	if err == nil {
		t.Fatalf("expected error for empty scenario body")
	}
}

// fakeAgent writes report.md and a screenshot into the work dir, simulating the
// verify-e2e skill driving a browser.
func fakeAgent(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only fake CLI")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-verify")
	script := `#!/usr/bin/env bash
work=""
for a in "$@"; do
  if [ -d "$a" ] && [ -f "$a/scenario.md" ]; then work="$a"; fi
done
if [ -z "$work" ]; then echo "no work dir with scenario.md" >&2; exit 1; fi
printf '# シナリオ\n\n## 判定\n- PASS\n' > "$work/report.md"
mkdir -p "$work/screenshots"
printf 'PNGDATA' > "$work/screenshots/01-top.png"
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	return bin
}

func TestRunScenarioEndToEnd(t *testing.T) {
	bin := fakeAgent(t)
	res, err := runScenario(context.Background(), Input{
		ScenarioTitle: "ログイン",
		ScenarioBody:  "# ログイン\n\n## 操作手順\n1. 開く",
		AppURL:        "http://localhost:3000",
	}, agent.Claude, bin, "", DefaultBrowserMCP(), nil)
	if err != nil {
		t.Fatalf("runScenario: %v", err)
	}
	if !strings.Contains(res.Report, "PASS") {
		t.Fatalf("unexpected report:\n%s", res.Report)
	}
	if len(res.Screenshots) != 1 || res.Screenshots[0].Name != "01-top.png" {
		t.Fatalf("expected one screenshot read back, got %+v", res.Screenshots)
	}
	if string(res.Screenshots[0].Data) != "PNGDATA" {
		t.Fatalf("screenshot data not read back")
	}
}

func TestRunScenarioReportsMissingReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "noop")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write noop: %v", err)
	}
	_, err := runScenario(context.Background(), Input{ScenarioBody: "x", AppURL: "http://x"}, agent.Codex, bin, "", DefaultBrowserMCP(), nil)
	if err == nil || !strings.Contains(err.Error(), "report.md") {
		t.Fatalf("expected missing-report error, got %v", err)
	}
}
