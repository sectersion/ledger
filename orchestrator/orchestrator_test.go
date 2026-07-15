package orchestrator

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sectersion/ledger/failure"
)

func toyRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	files := map[string]string{
		"go.mod":       "module toy\n\ngo 1.21\n",
		"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-m", "init")
	return repo
}

func autoApprove(o *Orchestrator) {
	go func() {
		for g := range o.Gates() {
			g.Approve()
		}
	}()
}

// TestRunPipelineDrivesAllPhases proves the orchestration machinery: phases
// run in order, gates fire and unblock on approval, and agents are tracked
// through to completion. It accepts either a fully successful run or a
// well-formed Validate escalation, rather than demanding one exact outcome
// — whether live models judge a plan "complied with" varies run to run;
// that judgment call belongs to Validate's own tests (phases package), not
// to this orchestration-wiring test.
func TestRunPipelineDrivesAllPhases(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	repo := toyRepo(t)
	journalPath := filepath.Join(repo, ".ledger", "orchestrator.jsonl")

	o := New()
	autoApprove(o)

	drained := make(chan struct{})
	go func() {
		for range o.Updates() {
		}
		close(drained)
	}()

	err := o.RunPipeline(context.Background(), repo, `Change the greeting main.go prints from "hello" to "hello, ledger".`, journalPath)
	if err != nil {
		var esc *failure.Escalation
		if !errors.As(err, &esc) || esc.Phase != "validate" {
			t.Fatalf("RunPipeline: unexpected error shape: %v", err)
		}
	}

	if phase := o.Phase(); phase != "done" && phase != "validate" {
		t.Fatalf("Phase() = %q, want done or validate", phase)
	}

	snap := o.Snapshot()
	if len(snap) == 0 {
		t.Fatal("expected at least one agent in the snapshot")
	}
	seenPhases := map[string]bool{}
	sawDone := false
	for _, a := range snap {
		seenPhases[a.Phase] = true
		if a.Status == StatusDone {
			sawDone = true
		}
	}
	for _, want := range []string{"research", "plan", "implement"} {
		if !seenPhases[want] {
			t.Errorf("expected at least one agent from phase %q", want)
		}
	}
	if !sawDone {
		t.Fatal("expected at least one agent to reach StatusDone")
	}
}

// TestRelayIsConsumedOnce proves the /btw plumbing worker.Run relies on:
// Relay queues a message under an agent ID, and the first TakeRelay for
// that ID returns it and clears it — so a respawn only picks up one relay
// per /btw press, not a repeat on every subsequent failure.
func TestRelayIsConsumedOnce(t *testing.T) {
	o := New()
	o.Relay("agent-1", "hold off on the DB migration")

	msg, ok := o.TakeRelay("agent-1")
	if !ok || msg != "hold off on the DB migration" {
		t.Fatalf("TakeRelay = %q, %v; want the queued message", msg, ok)
	}

	if _, ok := o.TakeRelay("agent-1"); ok {
		t.Fatal("TakeRelay should not return a message twice")
	}
	if _, ok := o.TakeRelay("agent-2"); ok {
		t.Fatal("TakeRelay should not return a message for an unrelated agent")
	}
}

func TestKillStopsAnInFlightAgent(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	repo := toyRepo(t)
	journalPath := filepath.Join(repo, ".ledger", "orchestrator.jsonl")

	o := New()
	autoApprove(o)

	killed := make(chan struct{})
	go func() {
		defer close(killed)
		var target string
		for u := range o.Updates() {
			// Target an actual research role, not the model-router call:
			// Choose() tolerates its own worker failing (falls back to the
			// first allowed model), so killing it wouldn't fail the pipeline.
			if strings.HasPrefix(u.AgentID, "ledger-research-") && target == "" {
				target = u.AgentID
				t.Logf("killing agent %q (phase %q)", target, u.Phase)
				o.Kill(target)
				return
			}
		}
	}()

	err := o.RunPipeline(context.Background(), repo, "toy repo sanity check", journalPath)
	if err == nil {
		t.Fatal("expected RunPipeline to fail once an in-flight agent is killed")
	}
	if !strings.Contains(err.Error(), "research") {
		t.Fatalf("expected the failure to come from the killed research role, got: %v", err)
	}

	select {
	case <-killed:
	case <-time.After(time.Second):
		t.Fatal("kill goroutine never observed an agent")
	}
}
