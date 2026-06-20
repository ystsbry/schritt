// Package tui implements schritt's Bubble Tea program for the refinement
// stage: paste a PBI, run the AI refinement, and browse the result sections.
//
// App is the root model. It owns the child views and the active-view state,
// drives the async refinement, and renders the global chrome (command line,
// status bar, help overlay).
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/refine"
	"github.com/ystsbry/schritt/internal/store"
	"github.com/ystsbry/schritt/internal/tui/keys"
	"github.com/ystsbry/schritt/internal/tui/views"
)

type viewState int

const (
	viewInput viewState = iota
	viewRunning
	viewList
	viewDetail
)

// Config carries the inputs NewApp needs from the caller.
type Config struct {
	// Refiner runs the AI refinement. Required unless Refinement is set
	// (view-only mode).
	Refiner refine.Refiner
	// Home is schritt's data directory (defaults applied by the caller).
	Home string
	// Model is recorded in the saved refinement's provenance. Optional.
	Model string
	// Refinement, when set, opens the app in view-only mode: it starts on the
	// result list showing this already-loaded refinement (and any
	// implement/verify reports), with no input/refine step.
	Refinement *model.Refinement
}

// refineDoneMsg is delivered when the async refinement finishes.
type refineDoneMsg struct {
	ref *model.Refinement
	err error
}

// App is the root Bubble Tea model.
type App struct {
	km      keys.KeyMap
	input   *views.Input
	list    *views.List
	detail  *views.Detail
	spinner spinner.Model
	state   viewState

	refiner  refine.Refiner
	home     string
	model    string
	ref      *model.Refinement
	pending  views.SubmitMsg
	viewOnly bool

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

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	a := &App{
		km:       km,
		input:    views.NewInput(km),
		list:     views.NewList(km),
		detail:   views.NewDetail(km),
		spinner:  sp,
		state:    viewInput,
		refiner:  cfg.Refiner,
		home:     cfg.Home,
		model:    cfg.Model,
		cmdInput: ti,
	}
	if cfg.Refinement != nil {
		// View-only mode: jump straight to the result list.
		a.viewOnly = true
		a.ref = cfg.Refinement
		a.list.SetRefinement(cfg.Refinement)
		a.detail.SetRefinement(cfg.Refinement)
		a.state = viewList
	}
	return a
}

func (a *App) Init() tea.Cmd { return textinput.Blink }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Reserve one row for the bottom command/status bar.
		a.forwardToActive(tea.WindowSizeMsg{Width: m.Width, Height: m.Height - 1})
		return a, nil

	case views.SubmitMsg:
		a.pending = m
		a.state = viewRunning
		a.clearStatus()
		return a, tea.Batch(a.spinner.Tick, a.runRefine(m))

	case refineDoneMsg:
		if m.err != nil {
			a.state = viewInput
			a.setError("refine: " + m.err.Error())
			return a, nil
		}
		a.ref = m.ref
		a.list.SetRefinement(m.ref)
		a.detail.SetRefinement(m.ref)
		a.state = viewList
		a.setInfo(fmt.Sprintf("リファインメント完了 (PBI #%d, %d セクション)", m.ref.PBI.Number, len(m.ref.Sections)))
		a.forwardSize()
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

	case spinner.TickMsg:
		if a.state != viewRunning {
			return a, nil
		}
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(m)
		return a, cmd

	case tea.KeyMsg:
		if a.cmdMode {
			return a.updateCommandMode(m)
		}
		return a.updateNormalMode(m)
	}

	return a.delegateToActive(msg)
}

func (a *App) updateNormalMode(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C always quits, even while typing in the input view.
	if key.Matches(m, a.km.Quit) {
		return a, tea.Quit
	}

	if a.showHelp {
		switch {
		case key.Matches(m, a.km.Help), m.Type == tea.KeyEsc:
			a.showHelp = false
		}
		return a, nil
	}

	switch a.state {
	case viewInput:
		// The input view owns every key (typing, Tab, Ctrl+R) — don't let
		// the global ":"/"?" bindings steal characters from the textarea.
		return a.delegateToActive(m)
	case viewRunning:
		// Ignore input while the refinement runs (Ctrl+C handled above).
		return a, nil
	}

	// List / detail states: global chrome is active.
	switch {
	case key.Matches(m, a.km.Help):
		a.showHelp = true
		return a, nil
	case key.Matches(m, a.km.Command):
		a.enterCommandMode()
		return a, textinput.Blink
	}
	return a.delegateToActive(m)
}

func (a *App) delegateToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch a.state {
	case viewInput:
		_, cmd := a.input.Update(msg)
		return a, cmd
	case viewDetail:
		_, cmd := a.detail.Update(msg)
		return a, cmd
	case viewList:
		_, cmd := a.list.Update(msg)
		return a, cmd
	default:
		return a, nil
	}
}

func (a *App) forwardToActive(msg tea.Msg) {
	a.input.Update(msg)
	a.list.Update(msg)
	a.detail.Update(msg)
}

func (a *App) forwardSize() {
	if a.width == 0 || a.height == 0 {
		return
	}
	a.forwardToActive(tea.WindowSizeMsg{Width: a.width, Height: a.height - 1})
}

// runRefine returns a command that performs the AI refinement, persists it,
// and loads it back — all off the UI goroutine.
func (a *App) runRefine(in views.SubmitMsg) tea.Cmd {
	refiner := a.refiner
	home := a.home
	mdl := a.model
	return func() tea.Msg {
		if refiner == nil {
			return refineDoneMsg{err: fmt.Errorf("no refiner configured")}
		}
		res, err := refiner.Refine(context.Background(), refine.Input{
			PBINumber: in.PBINumber,
			PBIBody:   in.PBIBody,
			Notes:     in.Notes,
			RepoPaths: in.RepoPaths,
		})
		if err != nil {
			return refineDoneMsg{err: err}
		}
		dir, err := store.Save(home, store.SaveInput{
			PBINumber: in.PBINumber,
			PBIBody:   in.PBIBody,
			Notes:     in.Notes,
			RepoPaths: in.RepoPaths,
			Result:    res,
			Model:     mdl,
			Now:       time.Now(),
		})
		if err != nil {
			return refineDoneMsg{err: err}
		}
		ref, err := store.Load(dir)
		if err != nil {
			return refineDoneMsg{err: err}
		}
		return refineDoneMsg{ref: ref}
	}
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

// runCommand dispatches a typed ":foo" command.
func (a *App) runCommand(input string) (tea.Model, tea.Cmd) {
	switch input {
	case "":
		return a, nil
	case "quit", "q":
		return a, tea.Quit
	case "help", "h":
		a.showHelp = true
		return a, nil
	case "new", "n":
		if a.viewOnly {
			a.setError("view モードでは :new は使えません")
			return a, nil
		}
		a.input.Reset()
		a.state = viewInput
		a.clearStatus()
		a.forwardSize()
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
	case viewInput:
		body = a.input.View()
	case viewRunning:
		body = a.runningView()
	case viewDetail:
		body = a.detail.View()
	default:
		body = a.list.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, a.bottomBar())
}

func (a *App) runningView() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).
		Render("schritt refinement")
	line := fmt.Sprintf("%s PBI #%d をリファインメント中…", a.spinner.View(), a.pending.PBINumber)
	hint := lipgloss.NewStyle().Faint(true).Render("refine-pbi skill を呼び出しています。完了までお待ちください。")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", line, "", hint)
}

func (a *App) bottomBar() string {
	if a.cmdMode {
		return a.cmdInput.View()
	}
	if a.statusMsg == "" {
		hint := "ctrl+c 終了"
		if a.state == viewList || a.state == viewDetail {
			hint = ": コマンド · ? ヘルプ · " + hint
		}
		return lipgloss.NewStyle().Faint(true).Render(hint)
	}
	if a.statusErr {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(a.statusMsg)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(a.statusMsg)
}

func (a *App) setInfo(s string)  { a.statusMsg = s; a.statusErr = false }
func (a *App) setError(s string) { a.statusMsg = s; a.statusErr = true }
func (a *App) clearStatus()      { a.statusMsg = ""; a.statusErr = false }

// State accessors exposed for tests (viewState is unexported).
func (a *App) IsInput() bool   { return a.state == viewInput }
func (a *App) IsRunning() bool { return a.state == viewRunning }
func (a *App) IsList() bool    { return a.state == viewList }
func (a *App) IsDetail() bool  { return a.state == viewDetail }

// Run starts the Bubble Tea program with the alt screen enabled.
func Run(cfg Config) error {
	p := tea.NewProgram(NewApp(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
