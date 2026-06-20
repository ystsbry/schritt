package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ystsbry/schritt/internal/refine"
)

func TestLatestRefinementDir(t *testing.T) {
	home := t.TempDir()
	mk := func(pbi int, now time.Time) string {
		dir, err := Save(home, SaveInput{
			PBINumber: pbi,
			PBIBody:   "# PBI",
			Result: refine.Result{
				Implementation: []refine.Doc{{File: "01.md", Title: "s", Body: "# s\n"}},
			},
			Now: now,
		})
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		return dir
	}
	_ = mk(7, time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC))
	newest := mk(7, time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC))

	got, err := LatestRefinementDir(home, 7)
	if err != nil {
		t.Fatalf("LatestRefinementDir: %v", err)
	}
	if got != newest {
		t.Fatalf("LatestRefinementDir = %q, want newest %q", got, newest)
	}
}

func TestLatestRefinementDirMissing(t *testing.T) {
	if _, err := LatestRefinementDir(t.TempDir(), 99); err == nil {
		t.Fatalf("expected error when no refinement exists")
	}
}

func TestReadPBI(t *testing.T) {
	home := t.TempDir()
	dir, err := Save(home, SaveInput{PBINumber: 1, PBIBody: "# 本文PBI", Now: time.Now()})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	body, err := ReadPBI(dir)
	if err != nil {
		t.Fatalf("ReadPBI: %v", err)
	}
	if body != "# 本文PBI" {
		t.Fatalf("ReadPBI = %q", body)
	}
	// Missing pbi.md → "" with no error.
	empty, err := ReadPBI(t.TempDir())
	if err != nil || empty != "" {
		t.Fatalf("ReadPBI(missing) = %q, %v", empty, err)
	}
}

func TestLoadIncludesReports(t *testing.T) {
	home := t.TempDir()
	dir, err := Save(home, SaveInput{
		PBINumber: 1, PBIBody: "# PBI",
		Result: refine.Result{
			Implementation: []refine.Doc{{File: "01-x.md", Title: "x", Body: "# x\n"}},
			Integration:    []refine.Doc{{File: "01-s.md", Title: "s", Body: "# s\n"}},
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := SaveReport(dir, "01-x.md", "# 実装レポート\n\n内容\n"); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}
	if _, err := SaveVerification(dir, "01-s.md", "# 検証レポート\n\nPASS\n", nil); err != nil {
		t.Fatalf("SaveVerification: %v", err)
	}

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.ImplementReports) != 1 || r.ImplementReports[0].Title != "実装レポート" {
		t.Fatalf("ImplementReports = %+v", r.ImplementReports)
	}
	if len(r.VerifyReports) != 1 || r.VerifyReports[0].Title != "検証レポート" {
		t.Fatalf("VerifyReports = %+v", r.VerifyReports)
	}
	// Entries: PO(1)+impl(1)+integ(1)+implReport(1)+verifyReport(1) = 5.
	if got := len(r.Entries()); got != 5 {
		t.Fatalf("expected 5 flattened entries, got %d", got)
	}
}

func TestReportNameAndSaveReport(t *testing.T) {
	if got := ReportName("implementation/01-design.md"); got != "01-design.md" {
		t.Fatalf("ReportName = %q", got)
	}
	dir := t.TempDir()
	path, err := SaveReport(dir, "01-design.md", "# レポート\n")
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}
	if filepath.Base(filepath.Dir(path)) != "reports" {
		t.Fatalf("report should live under reports/, got %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "# レポート\n" {
		t.Fatalf("report content = %q, %v", string(b), err)
	}
}
