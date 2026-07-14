// Package worktree wraps `git worktree` for creating and pruning the
// isolated checkouts each worker runs in.
package worktree

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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

// Worktree is one entry from `git worktree list`.
type Worktree struct {
	Path   string
	Branch string
}

// List returns every worktree under <repo>/.ledger/worktrees, the ones
// ledger creates and is safe to prune explicitly.
func List(repo string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	ledgerRoot := filepath.ToSlash(filepath.Join(repo, ".ledger", "worktrees"))
	var worktrees []Worktree
	var cur Worktree
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
			if strings.HasPrefix(filepath.ToSlash(cur.Path), ledgerRoot) {
				worktrees = append(worktrees, cur)
			}
		}
	}
	return worktrees, nil
}

// PruneAll removes every ledger-created worktree (and its branch) under
// <repo>/.ledger/worktrees, for the explicit `ledger prune` command.
func PruneAll(repo string) error {
	worktrees, err := List(repo)
	if err != nil {
		return err
	}
	for _, wt := range worktrees {
		if err := PruneWorktree(repo, wt.Path, wt.Branch); err != nil {
			return fmt.Errorf("prune %s: %w", wt.Path, err)
		}
	}
	return nil
}
