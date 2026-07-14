package failure

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestRetrySucceedsWithinCap(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")

	calls := 0
	err := Retry(context.Background(), journalPath, "implement", DefaultRetries, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient failure")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (succeeds on the last allowed attempt)", calls)
	}
}

func TestRetryEscalatesAfterCap(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")

	calls := 0
	err := Retry(context.Background(), journalPath, "implement", DefaultRetries, func(ctx context.Context) error {
		calls++
		return errors.New("permanent failure")
	})
	if err == nil {
		t.Fatal("expected an error after exhausting retries")
	}
	var esc *Escalation
	if !errors.As(err, &esc) {
		t.Fatalf("expected an *Escalation, got %T: %v", err, err)
	}
	if esc.Attempts != DefaultRetries+1 {
		t.Fatalf("Attempts = %d, want %d", esc.Attempts, DefaultRetries+1)
	}
	if calls != DefaultRetries+1 {
		t.Fatalf("calls = %d, want %d", calls, DefaultRetries+1)
	}
}

func TestPrependMessage(t *testing.T) {
	if got := PrependMessage("", "task"); got != "task" {
		t.Fatalf("empty message: got %q, want %q", got, "task")
	}
	if got := PrependMessage("btw do X first", "task"); got != "btw do X first\n\ntask" {
		t.Fatalf("got %q", got)
	}
}
