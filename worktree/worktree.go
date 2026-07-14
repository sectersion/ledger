// Package worktree wraps `git worktree` for creating and pruning the
// isolated checkouts each worker runs in.
package worktree

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// CreateWorktree adds a new worktree for branch, rooted under
// <repo>/.ledger/worktrees/<branch>, creating the branch if it doesn't exist.
func CreateWorktree(repo, branch string) (string, error) {
	path := filepath.Join(repo, ".ledger", "worktrees", branch)
	cmd := exec.Command("git", "worktree", "add", "-b", branch, path)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return path, nil
}

// PruneWorktree removes the worktree at path and deletes branch, so the
// branch name is free for a later CreateWorktree call (e.g. a scoped
// re-run of the same role).
func PruneWorktree(repo, path, branch string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, out)
	}
	cmd = exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -D: %w: %s", err, out)
	}
	return nil
}
