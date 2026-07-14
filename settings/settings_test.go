package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(s, Defaults()) {
		t.Fatalf("got %+v, want defaults %+v", s, Defaults())
	}
}

func TestLoadPartialFileFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"concurrencyCap": 3}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.ConcurrencyCap != 3 {
		t.Fatalf("ConcurrencyCap = %d, want 3", s.ConcurrencyCap)
	}
	if len(s.ModelAllowList) == 0 {
		t.Fatal("expected ModelAllowList to fall back to defaults")
	}
	if s.MaxThinkingLevel != Defaults().MaxThinkingLevel {
		t.Fatalf("MaxThinkingLevel = %q, want default", s.MaxThinkingLevel)
	}
}
