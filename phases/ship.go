package phases

import (
	"context"
	"fmt"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/worker"
)

const shipPromptTmpl = `Commit the current working tree's changes with a clear commit message
summarizing what changed and why. Then try "gh pr create" to open a pull
request. If that fails (no gh, no remote, not authenticated), fall back to
"git push" and report the branch name for manual handoff. Report exactly
what you did and its outcome.`

// Ship delegates committing and shipping to a headless Claude Code
// instance in dir: try `gh pr create`, fall back to `git push`/branch
// handoff. It returns the worker's report of what happened.
func Ship(ctx context.Context, dir, journalPath string) (string, error) {
	report, err := worker.Run(worker.WithAgentID(ctx, "ship"), dir, shipPromptTmpl, "--allowed-tools", "Bash(git:*),Bash(gh:*)")
	if err != nil {
		journal.Append(journalPath, "error", map[string]string{"role": "ship", "error": err.Error()})
		return "", fmt.Errorf("ship: %w", err)
	}

	journal.Append(journalPath, "phase", map[string]string{"phase": "ship", "status": "done", "report": report})
	return report, nil
}
