// Package refine is the AI boundary of schritt's refinement stage. A Refiner
// takes a PBI and returns the four refinement sections as markdown. The
// concrete ClaudeRefiner shells out to the `claude` CLI; DemoRefiner returns
// canned content so the TUI can be exercised without an AI call.
//
// Keeping this an interface means the TUI and store layers never depend on
// how the sections are produced — only on the produced text.
package refine

import "context"

// Input is the PBI to refine.
type Input struct {
	// PBINumber is the product backlog item number. Required, positive.
	PBINumber int
	// PBIBody is the raw markdown of the PBI, as pasted by the user.
	PBIBody string
	// Notes is optional supplementary context — e.g. decisions or open
	// points from the refinement meeting — for the AI to take into account.
	// Empty when the user provided none.
	Notes string
	// RepoPaths are the optional paths to the target repositories. When set,
	// the skill is granted read access to each and asked to consult the
	// codebases so the implementation plan and test cases are concrete. Nil or
	// empty when the user provided none.
	RepoPaths []string
}

// Result holds the refinement sections as markdown. The single-file sections
// are plain strings; the implementation plan is an ordered list of steps, each
// its own markdown file.
type Result struct {
	POQuestions      string               // POへの確認事項
	Implementation   []ImplementationStep // 実装内容（ステップごと）
	UnitTests        string               // 単体テストのテストケース
	IntegrationTests string               // 統合テストのテストケース
}

// ImplementationStep is one step of the implementation plan.
type ImplementationStep struct {
	// File is the step's source filename within the implementation directory
	// (e.g. "01-setup.md"). Determines order via lexical sort.
	File string
	// Title is a human-facing label, derived from the first markdown heading
	// (falling back to the filename stem).
	Title string
	// Body is the step's markdown content.
	Body string
}

// Refiner turns a PBI into refinement sections.
type Refiner interface {
	// Refine runs the refinement. Implementations should honour ctx for
	// cancellation (e.g. the user quitting mid-run).
	Refine(ctx context.Context, in Input) (Result, error)
}
