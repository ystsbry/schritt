package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ystsbry/schritt/internal/model"
)

func newTestApp() *App {
	return NewApp(Config{Items: model.SampleItems()})
}

// send pushes a message through Update and returns the app. Any command the
// update emits is executed and the resulting message fed back, mirroring what
// the Bubble Tea runtime does — navigation relies on those follow-up messages.
func send(t *testing.T, a *App, msg tea.Msg) *App {
	t.Helper()
	m, cmd := a.Update(msg)
	got, ok := m.(*App)
	if !ok {
		t.Fatalf("Update returned %T, want *App", m)
	}
	if cmd != nil {
		if out := cmd(); out != nil {
			return send(t, got, out)
		}
	}
	return got
}

func TestEnterOpensDetail(t *testing.T) {
	a := newTestApp()
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !a.IsList() {
		t.Fatalf("expected to start in list view")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if !a.IsDetail() {
		t.Fatalf("expected detail view after Enter")
	}
}

func TestBackReturnsToList(t *testing.T) {
	a := newTestApp()
	a = send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !a.IsList() {
		t.Fatalf("expected list view after back")
	}
}

func TestUnknownCommandSetsError(t *testing.T) {
	a := newTestApp()
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	if !a.cmdMode {
		t.Fatalf("expected to enter command mode")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("bogus")})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if !a.statusErr {
		t.Fatalf("expected an error status for an unknown command")
	}
}
