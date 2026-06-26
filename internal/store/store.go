// Package store persists and loads refinements on disk. The layout mirrors
// revu: a refinement.yml manages metadata and references markdown body files
// that live alongside it.
//
//	~/.schritt/pbi-{N}/{timestamp}/
//	  refinement.yml
//	  pbi.md                 (the input, kept for reference)
//	  po_questions/      (one markdown file per confirmation item)
//	  implementation/    (one markdown file per step)
//	  integration_tests/ (one markdown file per E2E scenario)
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/refine"
)

// Home returns schritt's home directory. Defaults to ~/.schritt; the
// SCHRITT_HOME env var overrides it (used in tests).
func Home() (string, error) {
	if v := os.Getenv("SCHRITT_HOME"); v != "" {
		return v, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(h, ".schritt"), nil
}

// Directory sections hold one markdown file per item (confirmation item / step
// / scenario). Keep in sync with the refine package and SKILL.md.
const (
	poQuestionsSubdir    = "po_questions"
	implementationSubdir = "implementation"
	integrationSubdir    = "integration_tests"
)

// dirSectionSpec ties each directory section's ID to its subdir and the Result
// docs that supply its files. Every section is now multi-file: the PO questions
// are split one file per confirmation item, mirroring how the implementation
// and integration sections split into steps/scenarios.
var dirSectionSpec = []struct {
	id     string
	subdir string
	docs   func(refine.Result) []refine.Doc
}{
	{model.SectionPOQuestions, poQuestionsSubdir, func(r refine.Result) []refine.Doc { return r.POQuestions }},
	{model.SectionImplementation, implementationSubdir, func(r refine.Result) []refine.Doc { return r.Implementation }},
	{model.SectionIntegrationTests, integrationSubdir, func(r refine.Result) []refine.Doc { return r.Integration }},
}

// SaveInput carries everything Save needs to write a refinement.
type SaveInput struct {
	PBINumber int
	PBITitle  string   // optional; derived from the PBI if known
	PBIBody   string   // raw PBI markdown, persisted as pbi.md
	Notes     string   // optional supplementary context, persisted as notes.md
	RepoPaths []string // optional target repository paths, recorded in refinement.yml
	Result    refine.Result
	Model     string // recorded under generated_by; optional
	Now       time.Time
}

// Save writes a new refinement under home and returns its directory. The
// directory name is timestamped so repeated refinements of the same PBI don't
// clobber each other.
func Save(home string, in SaveInput) (string, error) {
	if home == "" {
		return "", errors.New("home is required")
	}
	if in.PBINumber <= 0 {
		return "", fmt.Errorf("PBINumber must be positive, got %d", in.PBINumber)
	}
	ts := in.Now.UTC().Format("20060102-150405")
	dir := filepath.Join(home, fmt.Sprintf("pbi-%d", in.PBINumber), ts)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	if in.PBIBody != "" {
		if err := os.WriteFile(filepath.Join(dir, "pbi.md"), []byte(in.PBIBody), 0o644); err != nil {
			return "", fmt.Errorf("write pbi.md: %w", err)
		}
	}
	if in.Notes != "" {
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(in.Notes), 0o644); err != nil {
			return "", fmt.Errorf("write notes.md: %w", err)
		}
	}

	byID := make(map[string]model.Section, len(model.SectionOrder))

	// Directory sections: one markdown file per item under a subdir.
	for _, spec := range dirSectionSpec {
		sec := model.Section{ID: spec.id, Title: model.SectionTitle[spec.id]}
		docs := spec.docs(in.Result)
		if len(docs) > 0 {
			if err := os.MkdirAll(filepath.Join(dir, spec.subdir), 0o755); err != nil {
				return "", fmt.Errorf("create %s: %w", spec.subdir, err)
			}
			for i, doc := range docs {
				name := doc.File
				if name == "" {
					name = fmt.Sprintf("%02d.md", i+1)
				}
				rel := spec.subdir + "/" + name
				if err := os.WriteFile(filepath.Join(dir, rel), []byte(doc.Body), 0o644); err != nil {
					return "", fmt.Errorf("write %s: %w", rel, err)
				}
				title := doc.Title
				if title == "" {
					title = strings.TrimSuffix(name, filepath.Ext(name))
				}
				sec.Steps = append(sec.Steps, model.Step{Title: title, BodyFile: rel})
			}
		}
		byID[spec.id] = sec
	}

	// Assemble sections in canonical order.
	sections := make([]model.Section, 0, len(model.SectionOrder))
	for _, id := range model.SectionOrder {
		sections = append(sections, byID[id])
	}

	r := model.Refinement{
		SchemaVersion: model.SchemaVersion,
		PBI:           model.PBIMeta{Number: in.PBINumber, Title: in.PBITitle},
		RepoPaths:     in.RepoPaths,
		GeneratedAt:   in.Now.UTC(),
		GeneratedBy:   model.GeneratedBy{Tool: "schritt", Model: in.Model},
		Sections:      sections,
	}
	out, err := yaml.Marshal(&r)
	if err != nil {
		return "", fmt.Errorf("marshal refinement: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "refinement.yml"), out, 0o644); err != nil {
		return "", fmt.Errorf("write refinement.yml: %w", err)
	}
	return dir, nil
}

// Load reads <dir>/refinement.yml and the referenced section bodies, returning
// a fully populated Refinement.
func Load(dir string) (*model.Refinement, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	yamlPath := filepath.Join(abs, "refinement.yml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", yamlPath, err)
	}
	var r model.Refinement
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", yamlPath, err)
	}
	if r.SchemaVersion != model.SchemaVersion {
		return nil, fmt.Errorf("%s: unsupported schema_version %d (expected %d)", yamlPath, r.SchemaVersion, model.SchemaVersion)
	}
	r.BaseDir = abs
	for i := range r.Sections {
		s := &r.Sections[i]
		// Multi-file (implementation) section: read each step body.
		if len(s.Steps) > 0 {
			for j := range s.Steps {
				st := &s.Steps[j]
				if st.BodyFile == "" {
					return nil, fmt.Errorf("%s: sections[%d].steps[%d].body_file is required", yamlPath, i, j)
				}
				body, err := os.ReadFile(filepath.Join(abs, st.BodyFile))
				if err != nil {
					return nil, fmt.Errorf("read step body %s: %w", st.BodyFile, err)
				}
				st.Body = string(body)
			}
			continue
		}
		// A section with neither steps nor a body file is an empty section
		// (e.g. a refinement saved with no documents for it). Leave Body empty
		// rather than erroring — every section is multi-file now, and an empty
		// one is harmless in the viewer.
		if s.BodyFile == "" {
			continue
		}
		// Legacy single-file section (refinements saved before the per-item
		// split stored the body in one file).
		body, err := os.ReadFile(filepath.Join(abs, s.BodyFile))
		if err != nil {
			return nil, fmt.Errorf("read section body %s: %w", s.BodyFile, err)
		}
		s.Body = string(body)
	}

	// Later-stage artifacts, when present.
	if reps, err := readReportDir(filepath.Join(abs, reportsSubdir)); err != nil {
		return nil, err
	} else {
		r.ImplementReports = reps
	}
	if reps, err := readReportDir(filepath.Join(abs, verificationSubdir)); err != nil {
		return nil, err
	} else {
		r.VerifyReports = reps
	}
	return &r, nil
}

// readReportDir reads the top-level *.md files in dir as ordered reports
// (sorted lexically so 01-, 02-, … keep their order). Subdirectories such as
// verification/screenshots/ are ignored. Returns nil when dir is absent.
func readReportDir(dir string) ([]model.Report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var reps []model.Report
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read report %s: %w", name, err)
		}
		body := string(raw)
		reps = append(reps, model.Report{Title: reportTitle(name, body), File: name, Body: body})
	}
	return reps, nil
}

// reportTitle derives a label from the first markdown heading, falling back to
// the filename stem.
func reportTitle(file, body string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	return strings.TrimSuffix(file, filepath.Ext(file))
}
