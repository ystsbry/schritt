// Package model holds the domain types the TUI renders. In a real tool you
// would replace Item (and SampleItems) with whatever data your app loads —
// the TUI in internal/tui only depends on these types, not on where they
// come from.
package model

// Item is a single row in the list view. Title is shown in the list; Body is
// rendered in the detail view.
type Item struct {
	Title string
	Body  string
}

// SampleItems returns placeholder data so the scaffold runs out of the box.
// Swap this out for your real data source (files, an API, git, …).
func SampleItems() []Item {
	return []Item{
		{
			Title: "Welcome to schritt",
			Body: "schritt is a starting point for a Bubble Tea TUI.\n\n" +
				"Use j/k to move, Enter to open an item, l to go back, and ? for help.\n" +
				"Press : to enter command mode, then type a command and Enter.",
		},
		{
			Title: "Where to start",
			Body: "internal/tui/app.go holds the root model (Init/Update/View).\n" +
				"internal/tui/views/ holds the individual screens.\n" +
				"internal/tui/keys/ centralises the key bindings.",
		},
		{
			Title: "Replace this data",
			Body: "These rows come from model.SampleItems(). Point cmd/schritt/main.go\n" +
				"at your real data and the rest of the UI stays the same.",
		},
	}
}
