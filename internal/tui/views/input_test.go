package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ystsbry/schritt/internal/tui/keys"
)

func TestInputTabCyclesFieldsBothWays(t *testing.T) {
	in := NewInput(keys.DefaultKeyMap())
	if in.focus != fieldNumber {
		t.Fatalf("expected initial focus on number, got %v", in.focus)
	}

	tab := tea.KeyMsg{Type: tea.KeyTab}
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}

	// Tab forward: number → repo → body → notes → number (wrap).
	for _, want := range []inputField{fieldRepo, fieldBody, fieldNotes, fieldNumber} {
		in, _ = in.Update(tab)
		if in.focus != want {
			t.Fatalf("after tab: focus = %v, want %v", in.focus, want)
		}
	}

	// Shift+Tab backward: number → notes (wrap) → body → repo → number.
	for _, want := range []inputField{fieldNotes, fieldBody, fieldRepo, fieldNumber} {
		in, _ = in.Update(shiftTab)
		if in.focus != want {
			t.Fatalf("after shift+tab: focus = %v, want %v", in.focus, want)
		}
	}
}
