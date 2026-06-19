package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui/keys"
)

// List is the top-level screen: a scrollable list of items.
type List struct {
	items  []model.Item
	km     keys.KeyMap
	cursor int
	width  int
	height int
}

func NewList(items []model.Item, km keys.KeyMap) *List {
	return &List{items: items, km: km}
}

// Cursor returns the index of the currently selected item.
func (l *List) Cursor() int { return l.cursor }

func (l *List) Update(msg tea.Msg) (*List, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = m.Width
		l.height = m.Height
		return l, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(m, l.km.Down):
			if l.cursor < len(l.items)-1 {
				l.cursor++
			}
		case key.Matches(m, l.km.Up):
			if l.cursor > 0 {
				l.cursor--
			}
		case key.Matches(m, l.km.Enter):
			if len(l.items) > 0 {
				return l, goToDetail(l.cursor)
			}
		}
	}
	return l, nil
}

func (l *List) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Render("schritt")

	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(l.items) == 0 {
		b.WriteString(faint.Render("(no items)"))
	}
	for i, it := range l.items {
		cursor := "  "
		line := it.Title
		if i == l.cursor {
			cursor = "> "
			line = selected.Render(line)
		}
		b.WriteString(cursor)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString(faint.Render("j/k move · enter open · ? help · q quit"))
	return b.String()
}
