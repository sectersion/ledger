package phases

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sectersion/ledger/registry"
)

func TestValidateDetectsFailingTestsAndScopesOwner(t *testing.T) {
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

	writeFiles := map[string]string{
		"go.mod":       "module toy\n\ngo 1.21\n",
		"main.go":      "package main\n\nfunc main() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestFails(t *testing.T) { t.Fatal(\"boom\") }\n",
	}
	for name, content := range writeFiles {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-m", "init")

	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")
	result, err := Validate(context.Background(), repo, "add a main function", journalPath)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed=false with a failing test")
	}
	if len(result.FailedPackages) != 1 {
		t.Fatalf("got %d failed packages, want 1: %v", len(result.FailedPackages), result.FailedPackages)
	}

	registryPath := filepath.Join(repo, ".ledger", "registry.json")
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	reg, err := registry.Load(registryPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := reg.Acquire(result.FailedPackages[0], "Backend"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	owners := FailingOwners(reg, result.FailedPackages)
	if len(owners) != 1 || owners[0] != "Backend" {
		t.Fatalf("FailingOwners = %v, want [Backend]", owners)
	}
}
