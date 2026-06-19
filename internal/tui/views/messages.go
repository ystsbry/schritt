package views

import tea "github.com/charmbracelet/bubbletea"

// GoToDetailMsg asks the app to switch to the detail view for Index.
type GoToDetailMsg struct{ Index int }

// GoToListMsg asks the app to switch back to the list view.
type GoToListMsg struct{}

func goToDetail(i int) tea.Cmd {
	return func() tea.Msg { return GoToDetailMsg{Index: i} }
}

func goToList() tea.Cmd {
	return func() tea.Msg { return GoToListMsg{} }
}
