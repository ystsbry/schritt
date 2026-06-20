// Package store persists and loads refinements on disk. The layout mirrors
// revu: a refinement.yml manages metadata and references markdown body files
// that live alongside it.
//
//	~/.schritt/pbi-{N}/{timestamp}/
//	  refinement.yml
//	  pbi.md                 (the input, kept for reference)
//	  po_questions.md
//	  implementation.md
//	  unit_tests.md
//	  integration_tests.md
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// sectionSpec ties a section ID to its body file and the Result field that
// supplies its content. The order here is the on-disk and display order.
var sectionSpec = []struct {
	id   string
	file string
	body func(refine.Result) string
}{
	{model.SectionPOQuestions, "po_questions.md", func(r refine.Result) string { return r.POQuestions }},
	{model.SectionImplementation, "implementation.md", func(r refine.Result) string { return r.Implementation }},
	{model.SectionUnitTests, "unit_tests.md", func(r refine.Result) string { return r.UnitTests }},
	{model.SectionIntegrationTests, "integration_tests.md", func(r refine.Result) string { return r.IntegrationTests }},
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

	sections := make([]model.Section, 0, len(sectionSpec))
	for _, spec := range sectionSpec {
		body := spec.body(in.Result)
		if err := os.WriteFile(filepath.Join(dir, spec.file), []byte(body), 0o644); err != nil {
			return "", fmt.Errorf("write %s: %w", spec.file, err)
		}
		sections = append(sections, model.Section{
			ID:       spec.id,
			Title:    model.SectionTitle[spec.id],
			BodyFile: spec.file,
		})
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
		if s.BodyFile == "" {
			return nil, fmt.Errorf("%s: sections[%d].body_file is required", yamlPath, i)
		}
		body, err := os.ReadFile(filepath.Join(abs, s.BodyFile))
		if err != nil {
			return nil, fmt.Errorf("read section body %s: %w", s.BodyFile, err)
		}
		s.Body = string(body)
	}
	return &r, nil
}
