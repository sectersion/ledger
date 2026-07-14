package worker

import (
	"context"
	"os/exec"
	"testing"
)

// Manual test per IMPLEMENTATION_PLAN.md M1: spawn one worker against a
// scratch dir and confirm events stream. Skips if `claude` isn't on PATH
// (e.g. CI) since this exercises the real CLI, not a mock.
func TestSpawnWorkerStreamsEvents(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	events, err := SpawnWorker(context.Background(), t.TempDir(), "reply with exactly the word ok")
	if err != nil {
		t.Fatalf("SpawnWorker: %v", err)
	}

	got := 0
	for range events {
		got++
	}
	if got == 0 {
		t.Fatal("expected at least one event, got none")
	}
}

// TestSpawnWorkerKilledByContext proves the M7 kill primitive: canceling
// ctx stops the worker rather than letting it run to completion.
func TestSpawnWorkerKilledByContext(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := SpawnWorker(ctx, t.TempDir(), "reply with exactly the word ok"); err == nil {
		t.Fatal("expected SpawnWorker to fail immediately on an already-canceled context")
	}
}
