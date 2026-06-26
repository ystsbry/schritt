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

// Result holds the refinement sections as markdown. Each section is an ordered
// list of documents, one markdown file per item: the PO questions are split one
// file per confirmation item, the implementation plan one file per step, and
// the integration (E2E) scenarios one file per scenario.
type Result struct {
	POQuestions    []Doc // POへの確認事項（確認事項ごと）
	Implementation []Doc // 実装内容（実装ステップごと）
	Integration    []Doc // 統合テスト（E2Eシナリオごと）
}

// Doc is one ordered markdown file within a multi-file section — a PO
// confirmation item, an implementation step, or an integration/E2E scenario.
type Doc struct {
	// File is the source filename within the section directory (e.g.
	// "01-setup.md"). Determines order via lexical sort.
	File string
	// Title is a human-facing label, derived from the first markdown heading
	// (falling back to the filename stem).
	Title string
	// Body is the document's markdown content.
	Body string
}

// Refiner turns a PBI into refinement sections.
type Refiner interface {
	// Refine runs the refinement. Implementations should honour ctx for
	// cancellation (e.g. the user quitting mid-run).
	//
	// progress, if non-nil, receives human-readable progress lines as the
	// refinement runs (the underlying AI CLI's tool calls and messages), so a
	// caller such as the TUI can surface live progress. It may be called from a
	// goroutine other than the caller's; implementations must not assume it is
	// safe to call after Refine returns.
	Refine(ctx context.Context, in Input, progress func(string)) (Result, error)
}
