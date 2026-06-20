package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/refine"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	in := SaveInput{
		PBINumber: 123,
		PBITitle:  "ログイン機能",
		PBIBody:   "# PBI\n\n本文",
		Result: refine.Result{
			POQuestions: "確認事項",
			Implementation: []refine.Doc{
				{File: "01-design.md", Title: "設計", Body: "# 設計\n\n方針\n"},
				{File: "02-build.md", Title: "実装", Body: "# 実装\n\n本体\n"},
			},
			UnitTests: "単体ケース",
			Integration: []refine.Doc{
				{File: "01-happy.md", Title: "正常系", Body: "# 正常系\n\nシナリオ\n"},
			},
		},
		Model: "demo",
		Now:   time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC),
	}

	dir, err := Save(home, in)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Directory layout: ~/pbi-123/{ts}/refinement.yml + single-file bodies +
	// the implementation/ and integration_tests/ directories.
	if got := filepath.Base(filepath.Dir(dir)); got != "pbi-123" {
		t.Fatalf("expected parent dir pbi-123, got %q", got)
	}
	for _, f := range []string{
		"refinement.yml", "pbi.md", "po_questions.md", "unit_tests.md",
		"implementation/01-design.md", "implementation/02-build.md",
		"integration_tests/01-happy.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
		}
	}
	// implementation.md / integration_tests.md must NOT exist (now directories).
	for _, f := range []string{"implementation.md", "integration_tests.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); !os.IsNotExist(err) {
			t.Fatalf("%s should not exist; got err=%v", f, err)
		}
	}

	r, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.SchemaVersion != model.SchemaVersion {
		t.Fatalf("schema_version = %d, want %d", r.SchemaVersion, model.SchemaVersion)
	}
	if r.PBI.Number != 123 || r.PBI.Title != "ログイン機能" {
		t.Fatalf("unexpected PBI meta: %+v", r.PBI)
	}
	if len(r.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(r.Sections))
	}
	// Sections come back in canonical order.
	wantIDs := model.SectionOrder
	for i, s := range r.Sections {
		if s.ID != wantIDs[i] {
			t.Fatalf("section[%d] id = %q, want %q", i, s.ID, wantIDs[i])
		}
		if s.Title != model.SectionTitle[s.ID] {
			t.Fatalf("section[%d] title = %q, want %q", i, s.Title, model.SectionTitle[s.ID])
		}
		switch s.ID {
		case model.SectionImplementation:
			if len(s.Steps) != 2 {
				t.Fatalf("implementation should have 2 steps, got %d", len(s.Steps))
			}
		case model.SectionIntegrationTests:
			if len(s.Steps) != 1 {
				t.Fatalf("integration should have 1 scenario, got %d", len(s.Steps))
			}
		default:
			if s.Body == "" {
				t.Fatalf("section[%d] (%s) has empty body", i, s.ID)
			}
		}
		for j, st := range s.Steps {
			if st.Title == "" || st.Body == "" || st.BodyFile == "" {
				t.Fatalf("section %s step[%d] incomplete: %+v", s.ID, j, st)
			}
		}
	}

	// Entries flatten: PO(1) + impl(2) + unit(1) + integ(1) = 5.
	if got := len(r.Entries()); got != 5 {
		t.Fatalf("expected 5 flattened entries, got %d", got)
	}
}

func TestSaveWritesNotesWhenPresent(t *testing.T) {
	home := t.TempDir()
	dir, err := Save(home, SaveInput{
		PBINumber: 5,
		PBIBody:   "# PBI",
		Notes:     "会議メモ: 認証はOAuthで合意",
		Now:       time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatalf("expected notes.md to exist: %v", err)
	}
	if string(got) != "会議メモ: 認証はOAuthで合意" {
		t.Fatalf("notes.md content = %q", string(got))
	}
}

func TestSaveOmitsNotesWhenEmpty(t *testing.T) {
	home := t.TempDir()
	dir, err := Save(home, SaveInput{PBINumber: 6, PBIBody: "# PBI", Now: time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no notes.md when notes empty, got err=%v", err)
	}
}

func TestSaveRejectsBadPBINumber(t *testing.T) {
	if _, err := Save(t.TempDir(), SaveInput{PBINumber: 0}); err == nil {
		t.Fatalf("expected error for PBINumber 0")
	}
}

func TestHomeEnvOverride(t *testing.T) {
	t.Setenv("SCHRITT_HOME", "/tmp/schritt-test-home")
	got, err := Home()
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if got != "/tmp/schritt-test-home" {
		t.Fatalf("Home() = %q, want override", got)
	}
}
