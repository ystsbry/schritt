# schritt

A starting point for a terminal UI built in Go with [Bubble Tea].

> *schritt* — German for "step". A small, well-structured base to build your TUI on, one step at a time.

## Quick start

```sh
make run        # build + launch the TUI
make build      # build bin/schritt
make test       # run tests
make lint       # golangci-lint (needs golangci-lint installed)
```

Or directly:

```sh
go run ./cmd/schritt           # launch the TUI
go run ./cmd/schritt version   # print version
```

## Keybindings

| Key            | Action                       |
| -------------- | ---------------------------- |
| `j` / `↓`      | move down                    |
| `k` / `↑`      | move up                      |
| `Enter`        | open the selected item       |
| `l` / `Esc`    | back to the list             |
| `:`            | command mode (`:q`, `:help`) |
| `?`            | toggle help                  |
| `q` / `Ctrl+C` | quit                         |

## Layout

```
cmd/schritt/main.go      Cobra entrypoint; wires data into the TUI and runs it.
internal/model/          Domain types the UI renders (replace SampleItems).
internal/tui/app.go      Root Bubble Tea model: state, command line, help.
internal/tui/help.go     Static help overlay.
internal/tui/keys/       Centralised key bindings.
internal/tui/views/      Individual screens (list, detail) + nav messages.
```

The TUI follows the standard Bubble Tea `Model` shape — each view implements
`Update` / `View`, and `App` delegates to the active view while owning the
global chrome (command line, status bar, help). Views ask the app to switch
screens by returning navigation messages (`views.GoToDetailMsg`,
`views.GoToListMsg`) rather than mutating shared state directly.

## Making it yours

1. Replace `model.Item` / `model.SampleItems()` with your real data and load
   it in `cmd/schritt/main.go`.
2. Add screens under `internal/tui/views/` and a corresponding `viewState`
   in `app.go`.
3. Add key bindings in `internal/tui/keys/keys.go` and document them in
   `help.go`.
4. Add `:` commands in the `runCommand` switch in `app.go`.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
