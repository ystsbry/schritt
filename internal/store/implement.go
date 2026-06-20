package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// reportsSubdir holds one per-step implementation report.
const reportsSubdir = "reports"

// LatestRefinementDir returns the most recent {timestamp} refinement directory
// under <home>/pbi-{pbi}/. Timestamps are formatted so lexical order is
// chronological, so the lexically largest entry is the newest.
func LatestRefinementDir(home string, pbi int) (string, error) {
	if pbi <= 0 {
		return "", fmt.Errorf("pbi must be positive, got %d", pbi)
	}
	parent := filepath.Join(home, fmt.Sprintf("pbi-%d", pbi))
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", parent, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no refinement found under %s", parent)
	}
	sort.Strings(dirs)
	return filepath.Join(parent, dirs[len(dirs)-1]), nil
}

// ReadPBI returns the saved pbi.md body for a refinement directory, or "" if it
// was not persisted.
func ReadPBI(dir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "pbi.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read pbi.md: %w", err)
	}
	return string(b), nil
}

// ReportName maps an implementation step's body file (e.g.
// "implementation/01-design.md") to its report filename ("01-design.md").
func ReportName(stepBodyFile string) string {
	return filepath.Base(stepBodyFile)
}

// SaveReport writes a per-step implementation report under <dir>/reports/<name>
// and returns the absolute path written.
func SaveReport(dir, name, body string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("report name is required")
	}
	reportsDir := filepath.Join(dir, reportsSubdir)
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", reportsDir, err)
	}
	path := filepath.Join(reportsDir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
