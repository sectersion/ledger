package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreateAndPruneWorktree(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")

	path, err := CreateWorktree(repo, "ledger-test-branch")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("worktree path missing: %v", err)
	}

	if err := PruneWorktree(repo, path, "ledger-test-branch"); err != nil {
		t.Fatalf("PruneWorktree: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree path still exists after prune")
	}
}

func TestListAndPruneAll(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")

	if _, err := CreateWorktree(repo, "ledger-list-a"); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if _, err := CreateWorktree(repo, "ledger-list-b"); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	worktrees, err := List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("got %d worktrees, want 2: %+v", len(worktrees), worktrees)
	}

	if err := PruneAll(repo); err != nil {
		t.Fatalf("PruneAll: %v", err)
	}
	remaining, err := List(repo)
	if err != nil {
		t.Fatalf("List after prune: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("got %d worktrees after PruneAll, want 0: %+v", len(remaining), remaining)
	}
}
