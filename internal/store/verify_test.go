package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveVerification(t *testing.T) {
	dir := t.TempDir()
	path, err := SaveVerification(dir, "01-login.md", "# ログイン\n\nPASS\n", []Screenshot{
		{Name: "01-top.png", Data: []byte("PNG1")},
		{Name: "02-after.png", Data: []byte("PNG2")},
	})
	if err != nil {
		t.Fatalf("SaveVerification: %v", err)
	}
	if filepath.Base(filepath.Dir(path)) != "verification" {
		t.Fatalf("report should live under verification/, got %q", path)
	}
	// Report body.
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "# ログイン\n\nPASS\n" {
		t.Fatalf("report content = %q, %v", string(b), err)
	}
	// Screenshots under verification/screenshots/<stem>/.
	shotDir := filepath.Join(dir, "verification", "screenshots", "01-login")
	for name, want := range map[string]string{"01-top.png": "PNG1", "02-after.png": "PNG2"} {
		got, err := os.ReadFile(filepath.Join(shotDir, name))
		if err != nil || string(got) != want {
			t.Fatalf("screenshot %s = %q, %v", name, string(got), err)
		}
	}
}

func TestSaveVerificationNoScreenshots(t *testing.T) {
	dir := t.TempDir()
	if _, err := SaveVerification(dir, "01-x.md", "report", nil); err != nil {
		t.Fatalf("SaveVerification: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "verification", "screenshots")); !os.IsNotExist(err) {
		t.Fatalf("screenshots dir should not exist with no shots, got err=%v", err)
	}
}
