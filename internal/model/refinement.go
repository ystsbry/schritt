// Package model holds the domain types schritt persists and renders. A
// Refinement is the output of the first pipeline stage: an AI pass over a PBI
// that produces questions for the PO, an implementation plan, and test cases.
//
// The on-disk format mirrors revu: a refinement.yml file manages metadata and
// references the section bodies, which live next to it as markdown files.
package model

import "time"

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
	SectionIntegrationTests: "統合テストのテストケース",
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

// Section is one part of the refinement result. The body is stored in a
// sibling markdown file referenced by BodyFile; Body itself is derived on
// load and not persisted in the YAML.
type Section struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	BodyFile string `yaml:"body_file"`

	// Derived (not persisted in YAML).
	Body string `yaml:"-"`
}

// Refinement is the full result loaded from a refinement.yml directory.
type Refinement struct {
	SchemaVersion int         `yaml:"schema_version"`
	PBI           PBIMeta     `yaml:"pbi"`
	GeneratedAt   time.Time   `yaml:"generated_at"`
	GeneratedBy   GeneratedBy `yaml:"generated_by"`
	Sections      []Section   `yaml:"sections"`

	// Derived (not persisted in YAML).
	BaseDir string `yaml:"-"`
}
