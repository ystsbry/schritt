package views

import tea "github.com/charmbracelet/bubbletea"

// SubmitMsg is emitted by the input view when the user submits a PBI for
// refinement. The app turns this into an AI call.
type SubmitMsg struct {
	PBINumber int
	PBIBody   string
	// Notes is optional supplementary context (e.g. what was discussed in
	// the refinement meeting) that the AI should take into account.
	Notes string
	// RepoPaths are the optional paths to the target repositories, each
	// resolved to an absolute path. When set, the AI consults the codebases.
	// Nil if none.
	RepoPaths []string
}

// GoToDetailMsg asks the app to switch to the detail view for Index.
type GoToDetailMsg struct{ Index int }

// GoToListMsg asks the app to switch back to the section list.
type GoToListMsg struct{}

func goToDetail(i int) tea.Cmd {
	return func() tea.Msg { return GoToDetailMsg{Index: i} }
}

func goToList() tea.Cmd {
	return func() tea.Msg { return GoToListMsg{} }
}
