// Package journal is the append-only, per-agent JSONL log that's the
// single source of truth for phase/status state on orchestrator restart.
package journal

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// Entry is one journal line. Kind identifies what it records (e.g. "diff",
// "verdict", "grant", "release", "error", "phase"); Data carries the
// kind-specific payload.
type Entry struct {
	Time time.Time       `json:"time"`
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// Append writes entry as one line to the journal file at path, creating it
// if needed.
func Append(path string, kind string, data any) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	line, err := json.Marshal(Entry{Time: time.Now(), Kind: kind, Data: raw})
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// Read loads every entry from the journal at path, in order. A missing
// file reads as an empty journal (nothing recorded yet).
func Read(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}
