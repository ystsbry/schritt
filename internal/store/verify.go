package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// verificationSubdir holds one per-scenario verification report plus a
// screenshots/ tree.
const verificationSubdir = "verification"

// Screenshot is one captured image to persist alongside a verification report.
type Screenshot struct {
	Name string // base filename, e.g. "01-top.png"
	Data []byte
}

// SaveVerification writes a per-scenario verification report under
// <dir>/verification/<scenarioFile> and its screenshots under
// <dir>/verification/screenshots/<scenario-stem>/. Returns the report path.
func SaveVerification(dir, scenarioFile, report string, shots []Screenshot) (string, error) {
	if strings.TrimSpace(scenarioFile) == "" {
		return "", fmt.Errorf("scenario file name is required")
	}
	vdir := filepath.Join(dir, verificationSubdir)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", vdir, err)
	}
	reportPath := filepath.Join(vdir, scenarioFile)
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", reportPath, err)
	}

	if len(shots) > 0 {
		stem := strings.TrimSuffix(scenarioFile, filepath.Ext(scenarioFile))
		shotDir := filepath.Join(vdir, "screenshots", stem)
		if err := os.MkdirAll(shotDir, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", shotDir, err)
		}
		for _, s := range shots {
			name := filepath.Base(s.Name)
			if err := os.WriteFile(filepath.Join(shotDir, name), s.Data, 0o644); err != nil {
				return "", fmt.Errorf("write screenshot %s: %w", name, err)
			}
		}
	}
	return reportPath, nil
}
