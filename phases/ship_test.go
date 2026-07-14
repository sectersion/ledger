package phases

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShipCommitsWithNoRemote(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}

	// No remote configured: gh pr create and git push should both fail,
	// proving the worker at least gets to (and reports) the fallback path
	// without this test performing any real network/push side effect.
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
	if err := os.WriteFile(filepath.Join(repo, "feature.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")
	report, err := Ship(context.Background(), repo, journalPath)
	if err != nil {
		t.Fatalf("Ship: %v", err)
	}
	if report == "" {
		t.Fatal("expected a non-empty ship report")
	}

	log, logErr := exec.Command("git", "-C", repo, "log", "--oneline").CombinedOutput()
	if logErr != nil {
		t.Fatalf("git log: %v: %s", logErr, log)
	}
	if strings.Count(string(log), "\n") < 2 {
		t.Fatalf("expected a new commit, git log:\n%s", log)
	}
}
