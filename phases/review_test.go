package phases

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sectersion/ledger/journal"
)

func TestReviewProducesReportPendingSignoff(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

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
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")

	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")
	result, err := Review(context.Background(), repo, "add a main function", journalPath)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if result.Report == "" {
		t.Fatal("expected a non-empty review report")
	}
	if result.Approved {
		t.Fatal("expected Approved=false before an explicit sign-off")
	}

	if err := Approve(journalPath, true); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	entries, err := journal.Read(journalPath)
	if err != nil {
		t.Fatalf("journal.Read: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected journal entries after Review+Approve")
	}
}
