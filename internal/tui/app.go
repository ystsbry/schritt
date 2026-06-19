// Package tui implements the Bubble Tea program: a small list/detail UI with
// a command line and a help overlay. App is the root model; the individual
// screens live in internal/tui/views.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui/keys"
	"github.com/ystsbry/schritt/internal/tui/views"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
)

// Config carries the inputs NewApp needs from the caller.
type Config struct {
	Items []model.Item
}

// App is the root Bubble Tea model. It owns the child views, the active-view
// state, and the global chrome (command line, status message, help overlay).
type App struct {
	km     keys.KeyMap
	list   *views.List
	detail *views.Detail
	state  viewState

	cmdMode   bool
	cmdInput  textinput.Model
	statusMsg string
	statusErr bool
	showHelp  bool

	width  int
	height int
}

func NewApp(cfg Config) *App {
	km := keys.DefaultKeyMap()
	ti := textinput.New()
	ti.Prompt = ":"
	ti.CharLimit = 64
	return &App{
		km:       km,
		list:     views.NewList(cfg.Items, km),
		detail:   views.NewDetail(cfg.Items, km),
		state:    viewList,
		cmdInput: ti,
	}
}

func (a *App) Init() tea.Cmd { return nil }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Reserve one row for the bottom command/status bar.
		a.forwardToActive(tea.WindowSizeMsg{Width: m.Width, Height: m.Height - 1})
		return a, nil

	case views.GoToDetailMsg:
		a.detail.SetIndex(m.Index)
		a.state = viewDetail
		a.forwardSize()
		return a, nil

	case views.GoToListMsg:
		a.state = viewList
		a.forwardSize()
		return a, nil

	case tea.KeyMsg:
		if a.cmdMode {
			return a.updateCommandMode(m)
		}
		return a.updateNormalMode(m)
	}

	return a.delegateToActive(msg)
}

func (a *App) updateNormalMode(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.showHelp {
		// Any of ?, Esc, q dismisses help; everything else is swallowed.
		switch {
		case key.Matches(m, a.km.Help), m.Type == tea.KeyEsc, key.Matches(m, a.km.Quit):
			a.showHelp = false
		}
		return a, nil
	}
	switch {
	case key.Matches(m, a.km.Help):
		a.showHelp = true
		return a, nil
	case key.Matches(m, a.km.Command):
		a.enterCommandMode()
		return a, textinput.Blink
	case key.Matches(m, a.km.Quit):
		// In the detail view 'q' is reserved for quitting; Back ('l'/Esc)
		// returns to the list.
		return a, tea.Quit
	}
	return a.delegateToActive(m)
}

func (a *App) delegateToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch a.state {
	case viewDetail:
		_, cmd := a.detail.Update(msg)
		return a, cmd
	default:
		_, cmd := a.list.Update(msg)
		return a, cmd
	}
}

func (a *App) forwardToActive(msg tea.Msg) {
	switch a.state {
	case viewDetail:
		a.detail.Update(msg)
	default:
		a.list.Update(msg)
	}
}

func (a *App) forwardSize() {
	if a.width == 0 || a.height == 0 {
		return
	}
	a.forwardToActive(tea.WindowSizeMsg{Width: a.width, Height: a.height - 1})
}

func (a *App) updateCommandMode(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.exitCommandMode()
		return a, nil
	case tea.KeyEnter:
		input := strings.TrimSpace(a.cmdInput.Value())
		a.exitCommandMode()
		return a.runCommand(input)
	}
	var cmd tea.Cmd
	a.cmdInput, cmd = a.cmdInput.Update(m)
	return a, cmd
}

func (a *App) enterCommandMode() {
	a.cmdMode = true
	a.cmdInput.SetValue("")
	a.cmdInput.Focus()
	a.clearStatus()
}

func (a *App) exitCommandMode() {
	a.cmdMode = false
	a.cmdInput.Blur()
	a.cmdInput.SetValue("")
}

// runCommand dispatches a typed ":foo" command. Extend the switch as you add
// commands to the tool.
func (a *App) runCommand(input string) (tea.Model, tea.Cmd) {
	switch input {
	case "":
		return a, nil
	case "quit", "q":
		return a, tea.Quit
	case "help", "h":
		a.showHelp = true
		return a, nil
	}
	a.setError(fmt.Sprintf("unknown command: %s", input))
	return a, nil
}

func (a *App) View() string {
	if a.showHelp {
		return helpView(a.width, a.height)
	}
	var body string
	switch a.state {
	case viewDetail:
		body = a.detail.View()
	default:
		body = a.list.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, a.bottomBar())
}

func (a *App) bottomBar() string {
	if a.cmdMode {
		return a.cmdInput.View()
	}
	if a.statusMsg == "" {
		return lipgloss.NewStyle().Faint(true).Render("press : for command, ? for help")
	}
	if a.statusErr {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(a.statusMsg)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(a.statusMsg)
}

func (a *App) setError(s string) { a.statusMsg = s; a.statusErr = true }
func (a *App) clearStatus()      { a.statusMsg = ""; a.statusErr = false }

// State accessors exposed for tests (viewState is unexported).
func (a *App) IsList() bool   { return a.state == viewList }
func (a *App) IsDetail() bool { return a.state == viewDetail }

// Run starts the Bubble Tea program with the alt screen enabled.
func Run(cfg Config) error {
	p := tea.NewProgram(NewApp(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
