// Package worker spawns headless `claude` CLI instances and streams their
// stream-json output as typed events.
package worker

import (
	"bufio"
	"encoding/json"
	"os/exec"
)

// Event is one stream-json line from `claude --output-format stream-json`.
// Only "type" is parsed eagerly; Raw holds the full line for callers that
// need type-specific fields (system/assistant/user/result).
type Event struct {
	Type string
	Raw  json.RawMessage
}

// SpawnWorker runs `claude -p <prompt> --output-format stream-json` in cwd
// and streams parsed events as they arrive on the returned channel. The
// channel is closed when the process's stdout ends (EOF or process exit).
func SpawnWorker(cwd, prompt string) (<-chan Event, error) {
	cmd := exec.Command("claude", "-p", prompt, "--output-format", "stream-json", "--verbose")
	cmd.Dir = cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	events := make(chan Event)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var head struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(line, &head); err != nil {
				continue
			}
			raw := make(json.RawMessage, len(line))
			copy(raw, line)
			events <- Event{Type: head.Type, Raw: raw}
		}
		cmd.Wait()
	}()

	return events, nil
}
