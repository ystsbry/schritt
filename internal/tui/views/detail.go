package views

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui/keys"
)

// Detail renders one refinement entry's markdown body inside a scrollable
// viewport. Entries are the flattened list (one per item — confirmation item,
// implementation step, or E2E scenario).
type Detail struct {
	entries  []model.Entry
	km       keys.KeyMap
	index    int
	viewport viewport.Model
	width    int
	ready    bool
}

func NewDetail(km keys.KeyMap) *Detail {
	return &Detail{km: km}
}

// SetRefinement swaps in the refinement to read entries from.
func (d *Detail) SetRefinement(r *model.Refinement) {
	if r != nil {
		d.entries = r.Entries()
	} else {
		d.entries = nil
	}
}

// SetIndex selects which entry to show and resets the scroll position.
func (d *Detail) SetIndex(i int) {
	d.index = i
	if d.ready {
		d.viewport.SetContent(d.content())
		d.viewport.GotoTop()
	}
}

func (d *Detail) entry() (model.Entry, bool) {
	if d.index < 0 || d.index >= len(d.entries) {
		return model.Entry{}, false
	}
	return d.entries[d.index], true
}

// content returns the selected entry's body wrapped to the viewport width so
// long lines (including no-space CJK text) don't get clipped on the right. The
// viewport itself does not wrap; we must do it before SetContent.
func (d *Detail) content() string {
	e, ok := d.entry()
	if !ok {
		return ""
	}
	if d.width <= 0 {
		return e.Body
	}
	return lipgloss.NewStyle().Width(d.width).Render(e.Body)
}

func (d *Detail) title() string {
	if e, ok := d.entry(); ok {
		return e.Title
	}
	return ""
}

func (d *Detail) Update(msg tea.Msg) (*Detail, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve two rows: title line + trailing blank line.
		h := m.Height - 2
		if h < 1 {
			h = 1
		}
		d.width = m.Width
		if !d.ready {
			d.viewport = viewport.New(m.Width, h)
			d.ready = true
		} else {
			d.viewport.Width = m.Width
			d.viewport.Height = h
		}
		// Re-wrap content for the (possibly new) width.
		d.viewport.SetContent(d.content())
		return d, nil
	case tea.KeyMsg:
		if key.Matches(m, d.km.Back) {
			return d, goToList()
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *Detail) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Render(d.title())

	body := ""
	if d.ready {
		body = d.viewport.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}
