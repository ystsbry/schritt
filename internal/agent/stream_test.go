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
	sid, err := relayProgress(strings.NewReader(transcript), func(s string) {
		got = append(got, s)
	})
	if err != nil {
		t.Fatalf("relayProgress: %v", err)
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

func TestStreamingFlagsOnlyForClaudeWithProgress(t *testing.T) {
	emit := func(string) {}

	withProgress := Spec{Engine: Claude, SkillName: "refine-pbi", WorkDir: "/work", Progress: emit}.Args()
	if !strings.Contains(strings.Join(withProgress, " "), "--output-format stream-json --verbose") {
		t.Errorf("claude + Progress should add stream-json flags, got %v", withProgress)
	}

	noProgress := Spec{Engine: Claude, SkillName: "refine-pbi", WorkDir: "/work"}.Args()
	if strings.Contains(strings.Join(noProgress, " "), "stream-json") {
		t.Errorf("claude without Progress should not add stream-json flags, got %v", noProgress)
	}

	codex := Spec{Engine: Codex, SkillName: "refine-pbi", WorkDir: "/work", Progress: emit}.Args()
	if strings.Contains(strings.Join(codex, " "), "stream-json") {
		t.Errorf("codex should never add stream-json flags, got %v", codex)
	}
}
