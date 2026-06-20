// Package model holds the domain types schritt persists and renders. A
// Refinement is the output of the first pipeline stage: an AI pass over a PBI
// that produces questions for the PO, an implementation plan, and test cases.
//
// The on-disk format mirrors revu: a refinement.yml file manages metadata and
// references the section bodies, which live next to it as markdown files.
package model

import (
	"fmt"
	"time"
)

// SchemaVersion is the refinement.yml schema version. Bump it (and the loader
// guard) on any breaking change to the YAML shape.
const SchemaVersion = 1

// Section IDs. These are the fixed, ordered sections every refinement
// produces. Kept as constants so the loader, saver, and TUI agree.
const (
	SectionPOQuestions      = "po_questions"
	SectionImplementation   = "implementation"
	SectionUnitTests        = "unit_tests"
	SectionIntegrationTests = "integration_tests"
)

// SectionTitle maps a section ID to its human-facing title (Japanese, to
// match the team's PBI workflow).
var SectionTitle = map[string]string{
	SectionPOQuestions:      "POへの確認事項",
	SectionImplementation:   "実装内容",
	SectionUnitTests:        "単体テストのテストケース",
	SectionIntegrationTests: "統合テスト（E2Eシナリオ）",
}

// SectionOrder is the canonical display order of the sections.
var SectionOrder = []string{
	SectionPOQuestions,
	SectionImplementation,
	SectionUnitTests,
	SectionIntegrationTests,
}

// PBIMeta identifies the PBI a refinement was generated from.
type PBIMeta struct {
	Number int    `yaml:"number"`
	Title  string `yaml:"title,omitempty"`
}

// GeneratedBy records what produced the refinement, for provenance.
type GeneratedBy struct {
	Tool  string `yaml:"tool"`
	Model string `yaml:"model,omitempty"`
}

// Step is one ordered part of a multi-file section. The implementation plan is
// split into steps, each stored as its own markdown file under an
// implementation/ directory and referenced by BodyFile.
type Step struct {
	Title    string `yaml:"title"`
	BodyFile string `yaml:"body_file"`

	// Derived (not persisted in YAML).
	Body string `yaml:"-"`
}

// Section is one part of the refinement result. Single-file sections (PO
// questions, unit/integration tests) carry BodyFile. The implementation
// section instead carries an ordered list of Steps, one markdown file each.
// Body / Step.Body are derived on load and not persisted in the YAML.
type Section struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	BodyFile string `yaml:"body_file,omitempty"`
	Steps    []Step `yaml:"steps,omitempty"`

	// Derived (not persisted in YAML).
	Body string `yaml:"-"`
}

// Entry is a flattened, viewable unit of a refinement: one selectable row in
// the TUI with a title and a markdown body. Single-file sections map to one
// entry; the implementation section expands to one entry per step.
type Entry struct {
	Title string
	Body  string
}

// Report is a per-step/per-scenario markdown report produced by a later
// pipeline stage (implement → reports/, verify → verification/). Not part of
// refinement.yml; loaded from disk when present.
type Report struct {
	Title string
	File  string // base filename, e.g. "01-design.md"
	Body  string
}

// Refinement is the full result loaded from a refinement.yml directory.
type Refinement struct {
	SchemaVersion int         `yaml:"schema_version"`
	PBI           PBIMeta     `yaml:"pbi"`
	RepoPaths     []string    `yaml:"repo_paths,omitempty"`
	GeneratedAt   time.Time   `yaml:"generated_at"`
	GeneratedBy   GeneratedBy `yaml:"generated_by"`
	Sections      []Section   `yaml:"sections"`

	// Derived (not persisted in YAML). Loaded from sibling directories when
	// the later stages have run.
	BaseDir          string   `yaml:"-"`
	ImplementReports []Report `yaml:"-"` // reports/ (実装レポート)
	VerifyReports    []Report `yaml:"-"` // verification/ (検証レポート)
}

// Entries flattens the refinement into the list of viewable rows, in section
// order. The implementation section expands to one entry per step so each
// implementation step is browsable on its own.
func (r *Refinement) Entries() []Entry {
	var es []Entry
	for _, s := range r.Sections {
		if len(s.Steps) > 0 {
			for i, st := range s.Steps {
				es = append(es, Entry{
					Title: fmt.Sprintf("%s ▸ %d. %s", s.Title, i+1, st.Title),
					Body:  st.Body,
				})
			}
			continue
		}
		es = append(es, Entry{Title: s.Title, Body: s.Body})
	}
	for i, rep := range r.ImplementReports {
		es = append(es, Entry{Title: fmt.Sprintf("実装レポート ▸ %d. %s", i+1, rep.Title), Body: rep.Body})
	}
	for i, rep := range r.VerifyReports {
		es = append(es, Entry{Title: fmt.Sprintf("検証レポート ▸ %d. %s", i+1, rep.Title), Body: rep.Body})
	}
	return es
}
