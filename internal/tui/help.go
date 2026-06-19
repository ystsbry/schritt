package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpView is the static help overlay. Keep its rows in sync with
// keys.DefaultKeyMap and the command switch in runCommand.
func helpView(width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Padding(0, 1).
		Render("schritt — keybindings")

	sections := []struct {
		heading string
		rows    [][2]string
	}{
		{
			"Navigation",
			[][2]string{
				{"j / ↓", "next item / line down"},
				{"k / ↑", "prev item / line up"},
				{"Enter", "open detail (from list)"},
				{"l / Esc", "back to list (from detail)"},
			},
		},
		{
			"Commands (after :)",
			[][2]string{
				{":quit / :q", "quit"},
				{":help / :h", "show this help"},
			},
		},
		{
			"Misc",
			[][2]string{
				{"?", "toggle this help"},
				{":", "command mode"},
				{"q", "quit"},
			},
		},
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	descStyle := lipgloss.NewStyle()
	headingStyle := lipgloss.NewStyle().Underline(true).Bold(true).Foreground(lipgloss.Color("33"))

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, sec := range sections {
		b.WriteString(headingStyle.Render(sec.heading))
		b.WriteByte('\n')
		for _, kv := range sec.rows {
			b.WriteString("  ")
			b.WriteString(keyStyle.Render(padRight(kv[0], 12)))
			b.WriteString(descStyle.Render(kv[1]))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("press ? or Esc to return"))

	innerW := width - 4
	if innerW < 30 {
		innerW = 30
	}
	innerH := height - 2
	if innerH < 10 {
		innerH = 10
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(innerW).
		Height(innerH).
		Render(b.String())
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
