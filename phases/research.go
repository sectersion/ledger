// Package phases implements the pipeline's fixed stages (research, plan,
// implement, validate, review, ship), each a fan-out of worker roles over
// a worktree, aggregated into one artifact file.
package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/queue"
	"github.com/sectersion/ledger/settings"
	"github.com/sectersion/ledger/worker"
	"github.com/sectersion/ledger/worktree"
)

// researchRoles are the fixed subagent roles fanned out for the Research
// phase (PLAN.md's "compression of truth" stage).
var researchRoles = []struct {
	name   string
	prompt string
}{
	{"Codebase Locator", "Locate the files and directories relevant to this task, and list them with one-line descriptions. Task: %s"},
	{"Codebase Analyzer", "Analyze how the relevant existing code works today: control flow, data flow, key types. Task: %s"},
	{"Pattern Finder", "Find existing patterns/conventions in this codebase that a change for this task should follow. Task: %s"},
	{"External Research", "Research any external libraries, APIs, or prior art relevant to this task. Task: %s"},
	{"Historical Context", "Check git history/commit messages for context on why the relevant code looks the way it does, relative to this task. Task: %s"},
}

// Research fans out the fixed research roles in parallel worktrees, then
// aggregates their output into a single report file under
// <repo>/.ledger/research-report.md. It returns the report's path.
func Research(ctx context.Context, repo, task, journalPath string) (string, error) {
	q := queue.New(settings.LoadDefault().Cap(len(researchRoles)))
	tasks := make(chan queue.Task, len(researchRoles))
	reports := make([]string, len(researchRoles))

	for i, role := range researchRoles {
		i, role := i, role
		branch := fmt.Sprintf("ledger-research-%d", i)
		tasks <- queue.Task{
			ID:    role.name,
			Phase: "research",
			Run: func(ctx context.Context) error {
				wt, err := worktree.CreateWorktree(repo, branch)
				if err != nil {
					return fmt.Errorf("%s: %w", role.name, err)
				}
				defer worktree.PruneWorktree(repo, wt, branch)

				out, err := worker.Run(ctx, wt, fmt.Sprintf(role.prompt, task))
				if err != nil {
					journal.Append(journalPath, "error", map[string]string{"role": role.name, "error": err.Error()})
					return fmt.Errorf("%s: %w", role.name, err)
				}
				journal.Append(journalPath, "research", map[string]string{"role": role.name, "output": out})
				reports[i] = fmt.Sprintf("## %s\n\n%s\n", role.name, out)
				return nil
			},
		}
	}
	close(tasks)

	for id, err := range q.Run(ctx, tasks) {
		if err != nil {
			return "", fmt.Errorf("research: %s: %w", id, err)
		}
	}

	reportDir := filepath.Join(repo, ".ledger")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(reportDir, "research-report.md")
	content := "# Research Report\n\n" + task + "\n\n" + strings.Join(reports, "\n")
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	journal.Append(journalPath, "phase", map[string]string{"phase": "research", "status": "done", "report": reportPath})
	return reportPath, nil
}
