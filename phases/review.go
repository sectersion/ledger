package phases

import (
	"context"
	"fmt"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/worker"
)

// ReviewResult is the automated code-review artifact. Approved is the
// human sign-off: it starts false (pending) and is only true once the
// gate is explicitly approved (the TUI's a/r/e keybindings, per M9 —
// this phase only produces what that gate blocks on).
type ReviewResult struct {
	Report   string
	Approved bool
}

const reviewPromptTmpl = `Review the current working tree's changes for quality and security: bugs,
missing error handling, injection/secrets risks, and deviations from
repository conventions. Produce a concise review report (findings, or "no
issues found").

Plan the changes were meant to implement:
%s`

// Review runs an automated security/quality review of dir's working tree
// against plan, and returns it pending human sign-off.
func Review(ctx context.Context, dir, plan, journalPath string) (ReviewResult, error) {
	report, err := worker.Run(dir, fmt.Sprintf(reviewPromptTmpl, plan))
	if err != nil {
		return ReviewResult{}, fmt.Errorf("review: %w", err)
	}

	journal.Append(journalPath, "phase", map[string]any{
		"phase":  "review",
		"status": "pending_signoff",
		"report": report,
	})
	return ReviewResult{Report: report}, nil
}

// Approve records the human sign-off gate's decision for a completed
// review, so it's durable in the journal (single source of truth on
// restart, per PLAN.md's resume section).
func Approve(journalPath string, approved bool) error {
	return journal.Append(journalPath, "phase", map[string]any{
		"phase":    "review",
		"status":   "signoff",
		"approved": approved,
	})
}
