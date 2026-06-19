package views

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui/keys"
)

// Detail renders the body of a single item inside a scrollable viewport.
type Detail struct {
	items    []model.Item
	km       keys.KeyMap
	index    int
	viewport viewport.Model
	ready    bool
}

func NewDetail(items []model.Item, km keys.KeyMap) *Detail {
	return &Detail{items: items, km: km}
}

// SetIndex selects which item to show and resets the scroll position.
func (d *Detail) SetIndex(i int) {
	d.index = i
	if d.ready {
		d.viewport.SetContent(d.content())
		d.viewport.GotoTop()
	}
}

func (d *Detail) content() string {
	if d.index < 0 || d.index >= len(d.items) {
		return ""
	}
	return d.items[d.index].Body
}

func (d *Detail) title() string {
	if d.index < 0 || d.index >= len(d.items) {
		return ""
	}
	return d.items[d.index].Title
}

func (d *Detail) Update(msg tea.Msg) (*Detail, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve two rows: title line + trailing blank line.
		h := m.Height - 2
		if h < 1 {
			h = 1
		}
		if !d.ready {
			d.viewport = viewport.New(m.Width, h)
			d.viewport.SetContent(d.content())
			d.ready = true
		} else {
			d.viewport.Width = m.Width
			d.viewport.Height = h
		}
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
