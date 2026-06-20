package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui/keys"
)

// List is the refinement result overview: a header identifying the PBI and a
// selectable list of the refinement entries (single-file sections plus one row
// per implementation step).
type List struct {
	ref     *model.Refinement
	entries []model.Entry
	km      keys.KeyMap
	cursor  int
	width   int
	height  int
}

func NewList(km keys.KeyMap) *List {
	return &List{km: km}
}

// SetRefinement swaps in the refinement to display and resets the cursor.
func (l *List) SetRefinement(r *model.Refinement) {
	l.ref = r
	if r != nil {
		l.entries = r.Entries()
	} else {
		l.entries = nil
	}
	l.cursor = 0
}

// Cursor returns the index of the selected entry.
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
			if l.cursor < len(l.entries)-1 {
				l.cursor++
			}
		case key.Matches(m, l.km.Up):
			if l.cursor > 0 {
				l.cursor--
			}
		case key.Matches(m, l.km.Enter):
			if len(l.entries) > 0 {
				return l, goToDetail(l.cursor)
			}
		}
	}
	return l, nil
}

func (l *List) View() string {
	header := "schritt refinement"
	if l.ref != nil {
		header = fmt.Sprintf("リファインメント結果 — PBI #%d", l.ref.PBI.Number)
		if l.ref.PBI.Title != "" {
			header += " " + l.ref.PBI.Title
		}
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(header)

	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(l.entries) == 0 {
		b.WriteString(faint.Render("(セクションがありません)"))
	}
	for i, e := range l.entries {
		cursor := "  "
		line := e.Title
		if i == l.cursor {
			cursor = "> "
			line = selected.Render(line)
		}
		b.WriteString(cursor)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString(faint.Render("j/k 移動 · enter 開く · ? ヘルプ · ctrl+c 終了"))
	return b.String()
}
