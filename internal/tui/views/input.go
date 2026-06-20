package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/tui/keys"
)

// inputField identifies which widget has focus. Tab cycles through them in
// declaration order.
type inputField int

const (
	fieldNumber inputField = iota
	fieldRepo
	fieldBody
	fieldNotes
	numFields // sentinel: count of fields, for cycling
)

// Input is the first refinement screen: enter a PBI number, optionally a target
// repository path, paste the PBI markdown, optionally add supplementary notes
// (e.g. from the refinement meeting), then submit (Ctrl+R) to run.
type Input struct {
	km     keys.KeyMap
	number textinput.Model
	repo   textinput.Model
	body   textarea.Model
	notes  textarea.Model
	focus  inputField
	errMsg string
	width  int
	height int
}

func NewInput(km keys.KeyMap) *Input {
	num := textinput.New()
	num.Prompt = "PBI #: "
	num.Placeholder = "123"
	num.CharLimit = 9
	num.Focus()

	repo := textinput.New()
	repo.Prompt = "リポジトリ: "
	repo.Placeholder = "対象リポジトリのパス（任意・複数はカンマ区切り）"

	body := textarea.New()
	body.Placeholder = "ここにPBIのマークダウンを貼り付け…"
	body.ShowLineNumbers = false

	notes := textarea.New()
	notes.Placeholder = "リファインメント会議で話した内容・前提・決定事項など（任意）…"
	notes.ShowLineNumbers = false

	return &Input{km: km, number: num, repo: repo, body: body, notes: notes, focus: fieldNumber}
}

// Reset clears the fields so the screen is fresh for a new PBI.
func (in *Input) Reset() {
	in.number.SetValue("")
	in.repo.SetValue("")
	in.body.Reset()
	in.notes.Reset()
	in.errMsg = ""
	in.setFocus(fieldNumber)
}

func (in *Input) setFocus(f inputField) {
	in.focus = f
	in.number.Blur()
	in.repo.Blur()
	in.body.Blur()
	in.notes.Blur()
	switch f {
	case fieldNumber:
		in.number.Focus()
	case fieldRepo:
		in.repo.Focus()
	case fieldBody:
		in.body.Focus()
	case fieldNotes:
		in.notes.Focus()
	}
}

func (in *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		in.width = m.Width
		in.height = m.Height
		in.number.Width = m.Width - len(in.number.Prompt) - 2
		in.repo.Width = m.Width - len(in.repo.Prompt) - 2
		in.body.SetWidth(m.Width - 2)
		in.notes.SetWidth(m.Width - 2)
		// Chrome: title, blank, number, repo, blank, body label, notes label,
		// error line, hint line. Split the rest between the two textareas,
		// giving the PBI body the larger share.
		avail := m.Height - 10
		if avail < 6 {
			avail = 6
		}
		bodyH := avail * 2 / 3
		notesH := avail - bodyH
		if bodyH < 3 {
			bodyH = 3
		}
		if notesH < 2 {
			notesH = 2
		}
		in.body.SetHeight(bodyH)
		in.notes.SetHeight(notesH)
		return in, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(m, in.km.Submit):
			return in, in.submit()
		case key.Matches(m, in.km.Tab):
			in.setFocus((in.focus + 1) % numFields)
			return in, nil
		case key.Matches(m, in.km.ShiftTab):
			in.setFocus((in.focus - 1 + numFields) % numFields)
			return in, nil
		}
	}

	var cmd tea.Cmd
	switch in.focus {
	case fieldNumber:
		in.number, cmd = in.number.Update(msg)
	case fieldRepo:
		in.repo, cmd = in.repo.Update(msg)
	case fieldBody:
		in.body, cmd = in.body.Update(msg)
	case fieldNotes:
		in.notes, cmd = in.notes.Update(msg)
	}
	return in, cmd
}

// submit validates the fields and, if valid, emits a SubmitMsg. Repo and notes
// are optional, so only the number and body are required. A given repo path
// must resolve to an existing directory.
func (in *Input) submit() tea.Cmd {
	numStr := strings.TrimSpace(in.number.Value())
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		in.errMsg = "PBI番号は正の整数で入力してください"
		in.setFocus(fieldNumber)
		return nil
	}
	repoPaths, err := in.resolveRepos()
	if err != nil {
		in.errMsg = err.Error()
		in.setFocus(fieldRepo)
		return nil
	}
	if strings.TrimSpace(in.body.Value()) == "" {
		in.errMsg = "PBIのマークダウンを貼り付けてください"
		in.setFocus(fieldBody)
		return nil
	}
	in.errMsg = ""
	return func() tea.Msg {
		return SubmitMsg{
			PBINumber: n,
			PBIBody:   in.body.Value(),
			Notes:     strings.TrimSpace(in.notes.Value()),
			RepoPaths: repoPaths,
		}
	}
}

// resolveRepos validates the optional repo field, returning the absolute paths
// (or nil when left blank). Multiple repositories are comma-separated. Each
// entry expands a leading "~" and must be an existing directory, so errors
// surface here rather than deep in the AI run.
func (in *Input) resolveRepos() ([]string, error) {
	raw := strings.TrimSpace(in.repo.Value())
	if raw == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if p == "~" || strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, strings.TrimPrefix(p, "~"))
			}
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("リポジトリのパスを解決できません: %v", err)
		}
		st, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("リポジトリのパスが見つかりません: %s", abs)
		}
		if !st.IsDir() {
			return nil, fmt.Errorf("リポジトリのパスはディレクトリではありません: %s", abs)
		}
		out = append(out, abs)
	}
	return out, nil
}

func (in *Input) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Render("schritt refinement — PBI入力")

	label := lipgloss.NewStyle().Faint(true)
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(in.number.View())
	b.WriteByte('\n')
	b.WriteString(in.repo.View())
	b.WriteString("\n\n")
	b.WriteString(label.Render("PBI (markdown):"))
	b.WriteByte('\n')
	b.WriteString(in.body.View())
	b.WriteString("\n")
	b.WriteString(label.Render("補足 (リファインメント会議のメモ等・任意):"))
	b.WriteByte('\n')
	b.WriteString(in.notes.View())
	b.WriteString("\n")

	if in.errMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(in.errMsg))
		b.WriteByte('\n')
	}
	b.WriteString(label.Render(fmt.Sprintf("tab/shift+tab フィールド移動 · %s リファインメント実行 · ctrl+c 終了", in.km.Submit.Keys()[0])))
	return b.String()
}
