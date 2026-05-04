package agent

import "testing"

func TestSanitizedHistoryDropsToolProtocolMessages(t *testing.T) {
	history := []message{
		{Role: "user", Content: "list my customers"},
		{Role: "assistant", ToolCalls: []toolCall{{ID: "call-1"}}},
		{Role: "tool", ToolCallID: "call-1", Name: "list_customers", Content: `{"customers":[]}`},
		{Role: "assistant", Content: "You do not have customers yet."},
	}

	got := sanitizedHistory(history)

	if len(got) != 2 {
		t.Fatalf("expected only user and final assistant messages, got %#v", got)
	}
	if got[0].Role != "user" || got[1].Role != "assistant" {
		t.Fatalf("unexpected sanitized history: %#v", got)
	}
	if got[1].Content == "" {
		t.Fatalf("expected final assistant content to be retained")
	}
}
