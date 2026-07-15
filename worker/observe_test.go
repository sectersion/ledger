package worker

import (
	"context"
	"os/exec"
	"testing"
)

func TestEventSummary(t *testing.T) {
	cases := []struct {
		name string
		e    Event
		want string
	}{
		{"assistant text", Event{Type: "assistant", Raw: []byte(`{"message":{"content":[{"type":"text","text":"hi there"}]}}`)}, "hi there"},
		{"assistant tool use", Event{Type: "assistant", Raw: []byte(`{"message":{"content":[{"type":"tool_use","name":"Bash"}]}}`)}, "→ Bash"},
		{"result ok", Event{Type: "result", Raw: []byte(`{"is_error":false,"result":"done"}`)}, "✓ done"},
		{"result error", Event{Type: "result", Raw: []byte(`{"is_error":true,"result":"boom"}`)}, "✗ boom"},
		{"system noise", Event{Type: "system", Raw: []byte(`{}`)}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.e.Summary(); got != c.want {
				t.Fatalf("Summary() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRunReportsToSinkAndRegistrar(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	var gotID string
	var lines []string
	ctx := WithAgentID(context.Background(), "test-agent")
	ctx = WithSink(ctx, func(agentID string, e Event) {
		gotID = agentID
		if s := e.Summary(); s != "" {
			lines = append(lines, s)
		}
	})

	var registeredID string
	var cancel context.CancelFunc
	ctx = WithRegistrar(ctx, func(agentID string, c context.CancelFunc) {
		registeredID = agentID
		cancel = c
	})

	out, err := Run(ctx, t.TempDir(), "reply with exactly the word ok")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty result")
	}
	if gotID != "test-agent" {
		t.Fatalf("sink agent ID = %q, want test-agent", gotID)
	}
	if registeredID != "test-agent" {
		t.Fatalf("registrar agent ID = %q, want test-agent", registeredID)
	}
	if cancel == nil {
		t.Fatal("expected a registered cancel func")
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one summarized line")
	}
}
