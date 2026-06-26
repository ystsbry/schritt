package agent

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// relayClaudeProgress reads stream-json events from r (the claude CLI's stdout
// when run with --output-format stream-json --verbose) and emits one compact,
// human-readable progress line per meaningful event via emit. It returns the
// session_id observed on the system/init event (empty if absent) and the
// scanner error, if any.
//
// Unknown / unparseable lines are silently skipped — claude may add new event
// types over time, and missing a line is preferable to crashing mid-run.
func relayClaudeProgress(r io.Reader, emit func(string)) (string, error) {
	sc := bufio.NewScanner(r)
	// Stream-json lines can be large (assistant messages embed markdown
	// bodies). Bump the scanner's max token size well past the default 64 KiB.
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var sessionID string
	for sc.Scan() {
		var ev claudeStreamEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type == "system" && ev.Subtype == "init" && ev.SessionID != "" {
			sessionID = ev.SessionID
		}
		renderClaudeEvent(emit, ev)
	}
	return sessionID, sc.Err()
}

type claudeStreamEvent struct {
	Type      string               `json:"type"`
	Subtype   string               `json:"subtype,omitempty"`
	SessionID string               `json:"session_id,omitempty"`
	Message   *claudeStreamMessage `json:"message,omitempty"`

	// result fields (only set on type=="result")
	DurationMs int  `json:"duration_ms,omitempty"`
	NumTurns   int  `json:"num_turns,omitempty"`
	IsError    bool `json:"is_error,omitempty"`
}

type claudeStreamMessage struct {
	Content []claudeStreamBlock `json:"content"`
}

type claudeStreamBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// renderClaudeEvent turns one stream event into zero or more progress lines,
// each passed to emit.
func renderClaudeEvent(emit func(string), ev claudeStreamEvent) {
	switch ev.Type {
	case "system":
		if ev.Subtype == "init" {
			emit("claude セッション開始")
		}
	case "assistant":
		if ev.Message == nil {
			return
		}
		for _, b := range ev.Message.Content {
			switch b.Type {
			case "tool_use":
				summary, detail := summarizeToolUse(b)
				if summary == "" {
					continue
				}
				if detail != "" {
					emit(summary + " — " + detail)
				} else {
					emit(summary)
				}
			case "text":
				if s := firstLine(b.Text); s != "" {
					emit(truncate(s, 100))
				}
			}
		}
	}
}

// summarizeToolUse returns (one-line summary, optional detail) for a Claude
// tool_use block. Falls back to the bare tool name when the input shape isn't
// recognised so the user still sees that something happened.
func summarizeToolUse(b claudeStreamBlock) (string, string) {
	var in map[string]any
	if len(b.Input) > 0 {
		_ = json.Unmarshal(b.Input, &in)
	}
	getStr := func(k string) string {
		v, _ := in[k].(string)
		return v
	}

	switch b.Name {
	case "Bash":
		return "Bash: " + truncate(getStr("command"), 100), getStr("description")
	case "Read":
		return "Read: " + shortPath(getStr("file_path")), ""
	case "Write":
		return "Write: " + shortPath(getStr("file_path")), ""
	case "Edit":
		return "Edit: " + shortPath(getStr("file_path")), ""
	case "MultiEdit":
		return "MultiEdit: " + shortPath(getStr("file_path")), ""
	case "NotebookEdit":
		return "NotebookEdit: " + shortPath(getStr("notebook_path")), ""
	case "Glob":
		return "Glob: " + getStr("pattern"), getStr("path")
	case "Grep":
		return "Grep: " + getStr("pattern"), getStr("path")
	case "WebFetch":
		return "WebFetch: " + getStr("url"), ""
	case "WebSearch":
		return "WebSearch: " + truncate(getStr("query"), 80), ""
	case "Task", "Agent":
		desc := getStr("description")
		if desc == "" {
			desc = getStr("subagent_type")
		}
		return "Agent: " + desc, ""
	case "TodoWrite":
		return "TodoWrite", ""
	case "Skill":
		return "Skill: " + getStr("skill"), ""
	case "":
		return "", ""
	default:
		return b.Name, ""
	}
}

// shortPath replaces $HOME with ~ so progress lines stay scannable. Shared by
// the Claude and Codex relays.
func shortPath(p string) string {
	if p == "" {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			return "~/" + rel
		}
	}
	return p
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	// Operate on runes so we don't slice mid-codepoint on multibyte text.
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
