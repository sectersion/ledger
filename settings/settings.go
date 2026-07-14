// Package settings loads ~/.ledger/settings.json: the concurrency cap,
// model allow-list, and max thinking level.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings are ledger's user-configurable knobs. Zero values are replaced
// with defaults by Load, so callers never see an empty ModelAllowList or
// a zero ConcurrencyCap.
type Settings struct {
	ConcurrencyCap   int      `json:"concurrencyCap"`
	ModelAllowList   []string `json:"modelAllowList"`
	MaxThinkingLevel string   `json:"maxThinkingLevel"`
}

// Defaults matches PLAN.md's stated default concurrency cap of 10.
func Defaults() Settings {
	return Settings{
		ConcurrencyCap:   10,
		ModelAllowList:   []string{"claude-sonnet-5", "claude-opus-4-8", "claude-haiku-4-5-20251001"},
		MaxThinkingLevel: "high",
	}
}

// Path returns the default settings file location, ~/.ledger/settings.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ledger", "settings.json"), nil
}

// Load reads settings from path, filling in defaults for any zero-valued
// field. A missing file loads as all-defaults.
func Load(path string) (Settings, error) {
	s := Defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}

	if s.ConcurrencyCap == 0 {
		s.ConcurrencyCap = Defaults().ConcurrencyCap
	}
	if len(s.ModelAllowList) == 0 {
		s.ModelAllowList = Defaults().ModelAllowList
	}
	if s.MaxThinkingLevel == "" {
		s.MaxThinkingLevel = Defaults().MaxThinkingLevel
	}
	return s, nil
}

// LoadDefault loads settings from the default path (~/.ledger/settings.json),
// falling back to Defaults() if that can't be determined or read.
func LoadDefault() Settings {
	path, err := Path()
	if err != nil {
		return Defaults()
	}
	s, err := Load(path)
	if err != nil {
		return Defaults()
	}
	return s
}

// Cap returns the smaller of want and the configured concurrency cap.
func (s Settings) Cap(want int) int {
	if want > s.ConcurrencyCap {
		return s.ConcurrencyCap
	}
	return want
}
