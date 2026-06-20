package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/refine"
	"github.com/ystsbry/schritt/internal/tui/views"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	// Home is a temp dir so Save/Load stay off the real ~/.schritt.
	return NewApp(Config{Refiner: refine.DemoRefiner{}, Home: t.TempDir()})
}

// send pushes a message through Update and drains the resulting command so
// follow-up messages (navigation, refine completion) are delivered, mirroring
// the Bubble Tea runtime.
func send(t *testing.T, a *App, msg tea.Msg) *App {
	t.Helper()
	m, cmd := a.Update(msg)
	got, ok := m.(*App)
	if !ok {
		t.Fatalf("Update returned %T, want *App", m)
	}
	return drain(t, got, cmd)
}

// drain executes cmd and applies whatever message it yields. It unwraps
// tea.Batch (used by the refine flow) and feeds a single spinner tick without
// following its next tick, which would otherwise loop forever in a test.
func drain(t *testing.T, a *App, cmd tea.Cmd) *App {
	t.Helper()
	if cmd == nil {
		return a
	}
	switch msg := cmd().(type) {
	case nil:
		return a
	case tea.BatchMsg:
		for _, c := range msg {
			a = drain(t, a, c)
		}
		return a
	case spinner.TickMsg:
		m, _ := a.Update(msg)
		return m.(*App)
	default:
		m, next := a.Update(msg)
		return drain(t, m.(*App), next)
	}
}

func TestViewOnlyStartsInList(t *testing.T) {
	ref := &model.Refinement{
		PBI:              model.PBIMeta{Number: 7},
		ImplementReports: []model.Report{{Title: "実装レポート", File: "01.md", Body: "# 実装レポート\n"}},
		VerifyReports:    []model.Report{{Title: "検証レポート", File: "01.md", Body: "# 検証レポート\n"}},
	}
	a := NewApp(Config{Refinement: ref})
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !a.IsList() {
		t.Fatalf("view-only app should start on the result list")
	}
	// Reports are browsable: open the first entry.
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if !a.IsDetail() {
		t.Fatalf("expected detail view after Enter in view-only mode")
	}
}

func TestStartsInInput(t *testing.T) {
	a := newTestApp(t)
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !a.IsInput() {
		t.Fatalf("expected to start in the input view")
	}
}

func TestSubmitRunsRefinementAndShowsResult(t *testing.T) {
	a := newTestApp(t)
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Simulate the input view emitting a submit. The demo refiner returns
	// canned sections, so the flow runs to completion synchronously here.
	a = send(t, a, views.SubmitMsg{PBINumber: 42, PBIBody: "# Sample PBI\n\nDo the thing."})
	if !a.IsList() {
		t.Fatalf("expected list (result) view after refinement, state=%v", a.state)
	}
	if a.ref == nil || a.ref.PBI.Number != 42 {
		t.Fatalf("expected loaded refinement for PBI #42, got %+v", a.ref)
	}
	if len(a.ref.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(a.ref.Sections))
	}
}

func TestOpenSectionAndBack(t *testing.T) {
	a := newTestApp(t)
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a = send(t, a, views.SubmitMsg{PBINumber: 7, PBIBody: "x"})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if !a.IsDetail() {
		t.Fatalf("expected detail view after Enter")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !a.IsList() {
		t.Fatalf("expected list view after back")
	}
}

func TestNewCommandReturnsToInput(t *testing.T) {
	a := newTestApp(t)
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a = send(t, a, views.SubmitMsg{PBINumber: 1, PBIBody: "x"})
	// Enter command mode and run :new.
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("new")})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if !a.IsInput() {
		t.Fatalf("expected input view after :new")
	}
}
