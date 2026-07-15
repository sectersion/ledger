package phases

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestImplementEnforcesOwnershipAcrossWorkers(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(repo, "shared.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")

	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")
	plan := `Call the request_ownership tool for path "shared.txt" exactly once, then
reply with only the exact text the tool returned, and nothing else. Do not
edit any files.`

	roles := defaultImplementRoles
	outputs, err := ImplementScoped(context.Background(), repo, plan, journalPath, roles)
	if err != nil {
		t.Fatalf("ImplementScoped: %v", err)
	}
	if len(outputs) != len(roles) {
		t.Fatalf("got %d outputs, want %d", len(outputs), len(roles))
	}

	granted, denied := 0, 0
	for role, out := range outputs {
		switch {
		case strings.Contains(out, "granted"):
			granted++
		case strings.Contains(out, "denied"):
			denied++
		default:
			t.Errorf("role %s: unexpected output: %q", role, out)
		}
	}
	if granted != 1 {
		t.Errorf("granted = %d, want exactly 1", granted)
	}
	if denied != len(roles)-1 {
		t.Errorf("denied = %d, want %d", denied, len(roles)-1)
	}
}
