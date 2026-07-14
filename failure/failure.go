// Package failure is the M7 failure-handling policy: kill a failed
// worker/team and respawn a fresh one, capped at a fixed number of
// retries, then escalate as a gate-like blocking notification. `/btw`
// reuses this same kill-and-respawn primitive (PrependMessage), not a
// separate mechanism.
package failure

import (
	"context"
	"fmt"

	"github.com/sectersion/ledger/journal"
)

// DefaultRetries is PLAN.md's cap: 2 retries (3 attempts total) before
// escalating.
const DefaultRetries = 2

// Escalation is returned once retries are exhausted. It's a distinct type
// so callers (eventually the TUI) can render it as a blocking
// notification rather than a generic error.
type Escalation struct {
	Phase    string
	Attempts int
	Cause    error
}

func (e *Escalation) Error() string {
	return fmt.Sprintf("%s: failed after %d attempts, escalating: %v", e.Phase, e.Attempts, e.Cause)
}

func (e *Escalation) Unwrap() error { return e.Cause }

// Retry runs run, and on failure kills it (by virtue of run owning a
// fresh ctx/worktree/team each call — "kill worker + context, respawn
// fresh team") and tries again, up to retries times. If every attempt
// fails, it returns an *Escalation wrapping the last error.
func Retry(ctx context.Context, journalPath, phase string, retries int, run func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= retries+1; attempt++ {
		if attempt > 1 {
			journal.Append(journalPath, "retry", map[string]any{"phase": phase, "attempt": attempt})
		}
		if lastErr = run(ctx); lastErr == nil {
			return nil
		}
		journal.Append(journalPath, "error", map[string]any{"phase": phase, "attempt": attempt, "error": lastErr.Error()})
	}

	esc := &Escalation{Phase: phase, Attempts: retries + 1, Cause: lastErr}
	journal.Append(journalPath, "escalation", map[string]any{"phase": phase, "attempts": esc.Attempts, "error": lastErr.Error()})
	return esc
}

// PrependMessage relays a `/btw` message into a task's prompt ahead of a
// kill-and-respawn, per PLAN.md: there's no live channel to inject a
// message into a running worker, so the message is prepended to the task
// context of its replacement instead.
func PrependMessage(message, prompt string) string {
	if message == "" {
		return prompt
	}
	return message + "\n\n" + prompt
}
