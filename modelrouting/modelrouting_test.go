package modelrouting

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sectersion/ledger/settings"
)

func withSettings(t *testing.T, s settings.Settings) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows os.UserHomeDir reads this

	dir := filepath.Join(home, ".ledger")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChooseSkipsCallWithSingleAllowedModel(t *testing.T) {
	withSettings(t, settings.Settings{
		ConcurrencyCap:   10,
		ModelAllowList:   []string{"claude-sonnet-5"},
		MaxThinkingLevel: "high",
	})

	// No claude binary needed on PATH: a single-entry allow-list is
	// returned directly, without spawning anything.
	model, err := Choose(context.Background(), t.TempDir(), "any job")
	if err != nil {
		t.Fatalf("Choose: %v", err)
	}
	if model != "claude-sonnet-5" {
		t.Fatalf("model = %q, want claude-sonnet-5", model)
	}
}

func TestChooseReturnsAnAllowedModel(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH")
	}
	withSettings(t, settings.Settings{
		ConcurrencyCap:   10,
		ModelAllowList:   []string{"claude-sonnet-5", "claude-haiku-4-5-20251001"},
		MaxThinkingLevel: "high",
	})

	model, err := Choose(context.Background(), t.TempDir(), "reply with exactly the word ok, a trivial one-line job")
	if err != nil {
		t.Fatalf("Choose: %v", err)
	}
	if model != "claude-sonnet-5" && model != "claude-haiku-4-5-20251001" {
		t.Fatalf("model = %q, not in allow-list", model)
	}
}

func TestArgs(t *testing.T) {
	got := Args("claude-sonnet-5")
	want := []string{"--model", "claude-sonnet-5"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}
