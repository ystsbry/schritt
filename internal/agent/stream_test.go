package agent

import (
	"strings"
	"testing"
)

// TestRelayProgressSummarizesEvents feeds a representative stream-json transcript
// (one JSON event per line, as `claude --output-format stream-json --verbose`
// emits) and asserts that each meaningful event becomes a compact progress line.
func TestRelayProgressSummarizesEvents(t *testing.T) {
	transcript := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"sess-123"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"PBIを確認します\n詳細は省略"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/work/pbi.md"}}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./...","description":"テスト実行"}}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/work/po_questions.md"}}]}}`,
		`not json — should be skipped`,
		`{"type":"result","subtype":"success","duration_ms":4200,"num_turns":7}`,
	}, "\n")

	var got []string
	sid, err := relayClaudeProgress(strings.NewReader(transcript), func(s string) {
		got = append(got, s)
	})
	if err != nil {
		t.Fatalf("relayClaudeProgress: %v", err)
	}
	if sid != "sess-123" {
		t.Fatalf("session id = %q, want sess-123", sid)
	}

	want := []string{
		"claude セッション開始",
		"PBIを確認します", // only the first line of multi-line text
		"Read: /work/pbi.md",
		"Bash: go test ./... — テスト実行",
		"Write: /work/po_questions.md",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStreamingFlagsByEngineAndProgress(t *testing.T) {
	emit := func(string) {}

	claudeOn := strings.Join(Spec{Engine: Claude, SkillName: "refine-pbi", WorkDir: "/work", Progress: emit}.Args(), " ")
	if !strings.Contains(claudeOn, "--output-format stream-json --verbose") {
		t.Errorf("claude + Progress should add stream-json flags, got %q", claudeOn)
	}

	claudeOff := strings.Join(Spec{Engine: Claude, SkillName: "refine-pbi", WorkDir: "/work"}.Args(), " ")
	if strings.Contains(claudeOff, "stream-json") {
		t.Errorf("claude without Progress should not add stream-json flags, got %q", claudeOff)
	}

	codexOn := strings.Join(Spec{Engine: Codex, SkillName: "refine-pbi", WorkDir: "/work", Progress: emit}.Args(), " ")
	if !strings.Contains(codexOn, "--json") {
		t.Errorf("codex + Progress should add --json, got %q", codexOn)
	}
	if strings.Contains(codexOn, "stream-json") {
		t.Errorf("codex should use --json, not stream-json, got %q", codexOn)
	}

	codexOff := strings.Join(Spec{Engine: Codex, SkillName: "refine-pbi", WorkDir: "/work"}.Args(), " ")
	if strings.Contains(codexOff, "--json") {
		t.Errorf("codex without Progress should not add --json, got %q", codexOff)
	}
}

// TestRelayCodexProgressSummarizesEvents feeds a representative `codex exec
// --json` transcript and asserts each meaningful item becomes a compact line.
func TestRelayCodexProgressSummarizesEvents(t *testing.T) {
	transcript := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thr-9"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"type":"reasoning","text":"考え中"}}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"PBIを確認します\n詳細は省略"}}`,
		`{"type":"item.completed","item":{"type":"file_read","path":"/work/pbi.md"}}`,
		`{"type":"item.completed","item":{"type":"command_executed","command":"go test ./...","exit_code":1}}`,
		`{"type":"item.completed","item":{"type":"file_write","path":"/work/po_questions.md"}}`,
		`oops not json`,
		`{"type":"turn.completed","usage":{"input_tokens":10}}`,
	}, "\n")

	var got []string
	sid, err := relayCodexProgress(strings.NewReader(transcript), func(s string) {
		got = append(got, s)
	})
	if err != nil {
		t.Fatalf("relayCodexProgress: %v", err)
	}
	if sid != "thr-9" {
		t.Fatalf("thread id = %q, want thr-9", sid)
	}

	want := []string{
		"codex セッション開始",
		"PBIを確認します", // reasoning is suppressed; only the first line of agent_message
		"Read: /work/pbi.md",
		"Bash: go test ./... — exit 1",
		"Write: /work/po_questions.md",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
