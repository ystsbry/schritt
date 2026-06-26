package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// relayCodexProgress reads JSONL events from r (the codex CLI's stdout when run
// with `codex exec --json`) and emits one compact, human-readable progress line
// per meaningful event via emit. It returns the thread_id observed on the
// `thread.started` event (empty if absent) and the scanner error, if any.
//
// Codex's event schema differs from Claude's stream-json, so it needs its own
// parser; the rendered lines are kept stylistically identical (Bash: …, Read:
// …, Write: …) so the TUI log reads the same regardless of engine.
//
// Unknown / unparseable lines are silently skipped — codex may add new event
// types over time, and missing a line is preferable to crashing mid-run.
func relayCodexProgress(r io.Reader, emit func(string)) (string, error) {
	sc := bufio.NewScanner(r)
	// Item payloads can embed large text blobs (agent_message bodies, command
	// stdout). Bump the scanner buffer well past the default 64 KiB.
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var sessionID string
	for sc.Scan() {
		var ev codexStreamEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type == "thread.started" && ev.ThreadID != "" {
			sessionID = ev.ThreadID
		}
		renderCodexEvent(emit, ev)
	}
	return sessionID, sc.Err()
}

type codexStreamEvent struct {
	Type     string           `json:"type"`
	ThreadID string           `json:"thread_id,omitempty"`
	Item     *codexStreamItem `json:"item,omitempty"`

	// Some codex builds attach a top-level error/message field on the
	// `error` event.
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type codexStreamItem struct {
	Type string `json:"type,omitempty"`

	// agent_message
	Text string `json:"text,omitempty"`

	// command_executed
	Command  string `json:"command,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`

	// file_edit / file_change / file_read / file_write variants
	Path string `json:"path,omitempty"`

	// web_search / web_fetch
	Query string `json:"query,omitempty"`
	URL   string `json:"url,omitempty"`

	// generic name field on tool-call style items
	Name string `json:"name,omitempty"`
}

// renderCodexEvent turns one codex event into zero or more progress lines, each
// passed to emit.
func renderCodexEvent(emit func(string), ev codexStreamEvent) {
	switch ev.Type {
	case "thread.started":
		emit("codex セッション開始")
	case "item.completed":
		if ev.Item == nil {
			return
		}
		summary, detail := summarizeCodexItem(*ev.Item)
		if summary == "" {
			return
		}
		if detail != "" {
			emit(summary + " — " + detail)
		} else {
			emit(summary)
		}
	case "error":
		msg := ev.Error
		if msg == "" {
			msg = ev.Message
		}
		if msg == "" {
			msg = "(no message)"
		}
		emit("codex エラー: " + truncate(msg, 200))
	}
}

// summarizeCodexItem returns (one-line summary, optional detail) for an
// item.completed payload. Falls back to the bare item type so the user still
// sees something happened when codex introduces a new item kind.
func summarizeCodexItem(it codexStreamItem) (string, string) {
	switch it.Type {
	case "agent_message":
		if s := firstLine(it.Text); s != "" {
			return truncate(s, 100), ""
		}
		return "", ""
	case "reasoning":
		// Internal monologue — keep it quiet.
		return "", ""
	case "command_executed":
		if it.Command == "" {
			return "Bash", ""
		}
		detail := ""
		if it.ExitCode != nil && *it.ExitCode != 0 {
			detail = fmt.Sprintf("exit %d", *it.ExitCode)
		}
		return "Bash: " + truncate(it.Command, 100), detail
	case "file_edit", "file_change", "patch_applied":
		return "Edit: " + shortPath(it.Path), ""
	case "file_read":
		return "Read: " + shortPath(it.Path), ""
	case "file_write":
		return "Write: " + shortPath(it.Path), ""
	case "web_search":
		return "WebSearch: " + truncate(it.Query, 80), ""
	case "web_fetch":
		return "WebFetch: " + it.URL, ""
	case "tool_call":
		name := it.Name
		if name == "" {
			name = "tool"
		}
		return "Tool: " + name, ""
	case "":
		return "", ""
	default:
		return it.Type, ""
	}
}
