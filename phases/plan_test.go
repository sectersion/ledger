package phases

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanAgainstToyRepo(t *testing.T) {
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
	reportPath, err := Research(context.Background(), repo, "add a hello world print to main.go", journalPath)
	if err != nil {
		t.Fatalf("Research: %v", err)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	planPath, err := Plan(context.Background(), repo, string(report), journalPath)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	for _, role := range planRoles {
		if !strings.Contains(string(data), role.name) {
			t.Errorf("plan missing section for %s", role.name)
		}
	}
}
