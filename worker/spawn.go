// Package worker spawns headless `claude` CLI instances and streams their
// stream-json output as typed events.
package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
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
// Canceling ctx kills the worker process (the M7 kill primitive).
func SpawnWorker(ctx context.Context, cwd, prompt string, extraArgs ...string) (<-chan Event, error) {
	args := append([]string{"-p", prompt, "--output-format", "stream-json", "--verbose"}, extraArgs...)
	cmd := exec.CommandContext(ctx, "claude", args...)
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

// resultEvent mirrors the fields SpawnWorker's caller needs from a
// stream-json "result" event.
type resultEvent struct {
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// Run spawns a worker and blocks until it finishes, returning the text of
// its final "result" event. If ctx carries a Sink (WithSink) and/or
// Registrar (WithRegistrar), Run reports every event to the sink and
// registers its own cancel func under its agent ID (WithAgentID, or
// filepath.Base(cwd) if unset) — the M9 TUI's live-status and kill
// primitives, with no changes needed at any existing call site.
func Run(ctx context.Context, cwd, prompt string, extraArgs ...string) (string, error) {
	id := idFromContext(ctx)
	if id == "" {
		id = filepath.Base(cwd)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if reg := registrarFromContext(ctx); reg != nil {
		reg(id, cancel)
	}
	sink := sinkFromContext(ctx)

	events, err := SpawnWorker(ctx, cwd, prompt, extraArgs...)
	if err != nil {
		return "", err
	}

	var res resultEvent
	found := false
	for e := range events {
		if sink != nil {
			sink(id, e)
		}
		if e.Type != "result" {
			continue
		}
		if err := json.Unmarshal(e.Raw, &res); err != nil {
			return "", err
		}
		found = true
	}
	if !found {
		return "", fmt.Errorf("worker: no result event in %s output", cwd)
	}
	if res.IsError {
		return "", fmt.Errorf("worker: %s", res.Result)
	}
	return res.Result, nil
}
