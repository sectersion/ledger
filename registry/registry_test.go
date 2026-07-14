package registry

import (
	"path/filepath"
	"testing"
)

func TestAcquireDeniesOverlap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	ok, err := r.Acquire("src/foo.go", "agent-a")
	if err != nil || !ok {
		t.Fatalf("first Acquire: ok=%v err=%v", ok, err)
	}

	ok, err = r.Acquire("src/foo.go", "agent-b")
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if ok {
		t.Fatal("expected second Acquire to be denied")
	}

	if err := r.Release("src/foo.go", "agent-a"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	ok, err = r.Acquire("src/foo.go", "agent-b")
	if err != nil || !ok {
		t.Fatalf("Acquire after release: ok=%v err=%v", ok, err)
	}
}

func TestPersistsAcrossLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := r.Acquire("src/foo.go", "agent-a"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	ok, err := reloaded.Acquire("src/foo.go", "agent-b")
	if err != nil {
		t.Fatalf("Acquire after reload: %v", err)
	}
	if ok {
		t.Fatal("expected lock to survive reload and deny agent-b")
	}
}
