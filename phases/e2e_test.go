package phases

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sectersion/ledger/failure"
	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/registry"
)

// TestEndToEndDryRun is M10's end-to-end dry run: a toy repo through all 6
// phases, with an induced failure at Validate, confirming both the scoped
// RPI re-run (Validate -> FailingOwners -> ImplementScoped, via
// failure.Retry) and resume-from-crash (the journal alone reconstructs
// how far the pipeline got, without re-running anything) work.
func TestEndToEndDryRun(t *testing.T) {
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

	files := map[string]string{
		"go.mod":       "module toy\n\ngo 1.21\n",
		"main.go":      "package main\n\nfunc main() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestBroken(t *testing.T) { t.Fatal(\"induced failure\") }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-m", "init")

	journalPath := filepath.Join(repo, ".ledger", "orchestrator.jsonl")
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	task := "toy repo sanity check"

	reportPath, err := Research(ctx, repo, task, journalPath)
	if err != nil {
		t.Fatalf("Research: %v", err)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	planPath, err := Plan(ctx, repo, string(report), journalPath)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	plan, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}

	if _, err := Implement(ctx, repo, string(plan), journalPath); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	result, err := Validate(ctx, repo, string(plan), journalPath)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Validate to fail against the induced-failure toy repo")
	}
	if len(result.FailedPackages) == 0 {
		t.Fatal("expected at least one failed package")
	}

	// Seed the registry as if Implement's Backend role owned the failing
	// package, so FailingOwners has something real to scope against.
	reg, err := registry.Load(filepath.Join(repo, ".ledger", "registry.json"))
	if err != nil {
		t.Fatalf("registry.Load: %v", err)
	}
	if _, err := reg.Acquire(result.FailedPackages[0], "Backend"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	owners := FailingOwners(reg, result.FailedPackages)
	if len(owners) != 1 || owners[0] != "Backend" {
		t.Fatalf("FailingOwners = %v, want [Backend]", owners)
	}

	// Scoped RPI re-run: kill+respawn just the owning team via
	// failure.Retry, not the whole pipeline.
	if err := failure.Retry(ctx, journalPath, "implement-scoped", 1, func(ctx context.Context) error {
		_, err := ImplementScoped(ctx, repo, string(plan), journalPath, owners)
		return err
	}); err != nil {
		t.Fatalf("scoped re-run: %v", err)
	}

	if _, err := Review(ctx, repo, string(plan), journalPath); err != nil {
		t.Fatalf("Review: %v", err)
	}
	if err := Approve(journalPath, true); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if _, err := Ship(ctx, repo, journalPath); err != nil {
		t.Fatalf("Ship: %v", err)
	}

	// Resume-from-crash: a fresh read of the journal (as a restarted
	// orchestrator would do) must show every phase reached, without
	// re-running any of them.
	entries, err := journal.Read(journalPath)
	if err != nil {
		t.Fatalf("journal.Read: %v", err)
	}
	seenPhase := map[string]bool{}
	for _, e := range entries {
		if e.Kind != "phase" {
			continue
		}
		var p struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal(e.Data, &p); err == nil {
			seenPhase[p.Phase] = true
		}
	}
	for _, phase := range []string{"research", "plan", "implement", "validate", "review", "ship"} {
		if !seenPhase[phase] {
			t.Errorf("journal missing a %q phase entry; resume-from-crash would lose this phase", phase)
		}
	}
}
