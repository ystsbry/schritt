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
}

// Result holds the four refinement sections as markdown. Each field is the
// body that will be written to its corresponding section file.
type Result struct {
	POQuestions      string // POへの確認事項
	Implementation   string // 実装内容
	UnitTests        string // 単体テストのテストケース
	IntegrationTests string // 統合テストのテストケース
}

// Refiner turns a PBI into refinement sections.
type Refiner interface {
	// Refine runs the refinement. Implementations should honour ctx for
	// cancellation (e.g. the user quitting mid-run).
	Refine(ctx context.Context, in Input) (Result, error)
}
