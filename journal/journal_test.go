package journal

import (
	"path/filepath"
	"testing"
)

func TestAppendAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.jsonl")

	if err := Append(path, "phase", map[string]string{"phase": "plan"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Append(path, "verdict", map[string]string{"result": "approved"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Kind != "phase" || entries[1].Kind != "verdict" {
		t.Fatalf("unexpected kinds: %+v", entries)
	}
}

func TestReadMissingFile(t *testing.T) {
	entries, err := Read(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries for missing file, got %+v", entries)
	}
}
