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
			POQuestions:      "確認事項",
			Implementation:   "実装方針",
			UnitTests:        "単体ケース",
			IntegrationTests: "統合ケース",
		},
		Model: "demo",
		Now:   time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC),
	}

	dir, err := Save(home, in)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Directory layout: ~/pbi-123/{ts}/refinement.yml + bodies.
	if got := filepath.Base(filepath.Dir(dir)); got != "pbi-123" {
		t.Fatalf("expected parent dir pbi-123, got %q", got)
	}
	for _, f := range []string{"refinement.yml", "pbi.md", "po_questions.md", "implementation.md", "unit_tests.md", "integration_tests.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
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
	// Sections come back in canonical order with bodies populated.
	wantIDs := model.SectionOrder
	for i, s := range r.Sections {
		if s.ID != wantIDs[i] {
			t.Fatalf("section[%d] id = %q, want %q", i, s.ID, wantIDs[i])
		}
		if s.Body == "" {
			t.Fatalf("section[%d] (%s) has empty body", i, s.ID)
		}
		if s.Title != model.SectionTitle[s.ID] {
			t.Fatalf("section[%d] title = %q, want %q", i, s.Title, model.SectionTitle[s.ID])
		}
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
