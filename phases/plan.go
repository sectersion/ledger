package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/queue"
	"github.com/sectersion/ledger/worker"
	"github.com/sectersion/ledger/worktree"
)

// planRoles are the fixed subagent roles fanned out for the Plan phase
// (PLAN.md's "compression of intent" stage).
var planRoles = []struct {
	name   string
	prompt string
}{
	{"Architecture Planner", "Given this research report, produce an implementation plan: what files change, in what order, and why.\n\nResearch report:\n%s"},
	{"Risk Analyzer", "Given this research report, identify the risks/edge cases an implementation plan for this task must account for.\n\nResearch report:\n%s"},
	{"Test Planner", "Given this research report, produce a test plan: what should be tested and what the pass/fail rubric is.\n\nResearch report:\n%s"},
}

// Plan fans out the fixed planning roles in parallel worktrees, then
// aggregates their output into a single spec+rubric file under
// <repo>/.ledger/plan.md. It returns the plan's path.
func Plan(ctx context.Context, repo, researchReport, journalPath string) (string, error) {
	q := queue.New(len(planRoles))
	tasks := make(chan queue.Task, len(planRoles))
	sections := make([]string, len(planRoles))

	for i, role := range planRoles {
		i, role := i, role
		branch := fmt.Sprintf("ledger-plan-%d", i)
		tasks <- queue.Task{
			ID:    role.name,
			Phase: "plan",
			Run: func(ctx context.Context) error {
				wt, err := worktree.CreateWorktree(repo, branch)
				if err != nil {
					return fmt.Errorf("%s: %w", role.name, err)
				}
				defer worktree.PruneWorktree(repo, wt, branch)

				out, err := worker.Run(ctx, wt, fmt.Sprintf(role.prompt, researchReport))
				if err != nil {
					journal.Append(journalPath, "error", map[string]string{"role": role.name, "error": err.Error()})
					return fmt.Errorf("%s: %w", role.name, err)
				}
				journal.Append(journalPath, "plan", map[string]string{"role": role.name, "output": out})
				sections[i] = fmt.Sprintf("## %s\n\n%s\n", role.name, out)
				return nil
			},
		}
	}
	close(tasks)

	for id, err := range q.Run(ctx, tasks) {
		if err != nil {
			return "", fmt.Errorf("plan: %s: %w", id, err)
		}
	}

	planDir := filepath.Join(repo, ".ledger")
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		return "", err
	}
	planPath := filepath.Join(planDir, "plan.md")
	content := "# Plan\n\n" + strings.Join(sections, "\n")
	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	journal.Append(journalPath, "phase", map[string]string{"phase": "plan", "status": "done", "plan": planPath})
	return planPath, nil
}
