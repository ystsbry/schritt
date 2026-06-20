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
		Render("schritt — キーバインド")

	sections := []struct {
		heading string
		rows    [][2]string
	}{
		{
			"PBI入力",
			[][2]string{
				{"tab / shift+tab", "PBI番号 / リポジトリ / 本文 / 補足 を前後に移動"},
				{"ctrl+r", "リファインメント実行"},
			},
		},
		{
			"結果・レポートの閲覧",
			[][2]string{
				{"j / ↓", "次の項目 / 行送り"},
				{"k / ↑", "前の項目 / 行戻し"},
				{"Enter", "項目を開く（セクション/実装・検証レポート）"},
				{"l / Esc", "一覧に戻る"},
			},
		},
		{
			"コマンド ( : の後)",
			[][2]string{
				{":new / :n", "新しいPBIを入力"},
				{":help / :h", "このヘルプ"},
				{":quit / :q", "終了"},
			},
		},
		{
			"その他",
			[][2]string{
				{"?", "このヘルプの表示/非表示"},
				{":", "コマンドモード"},
				{"ctrl+c", "終了"},
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
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("? または Esc で戻る"))

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
