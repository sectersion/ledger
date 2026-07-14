package phases

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/modelrouting"
	"github.com/sectersion/ledger/ownership"
	"github.com/sectersion/ledger/queue"
	"github.com/sectersion/ledger/registry"
	"github.com/sectersion/ledger/settings"
	"github.com/sectersion/ledger/worker"
	"github.com/sectersion/ledger/worktree"
)

// implementRoles are the fixed worker roles fanned out for the Implement
// phase (PLAN.md's "mechanical execution" stage). Each must request
// ownership of a path from the orchestrator before editing it.
var implementRoles = []string{"Backend", "Frontend", "Test"}

const implementPromptTmpl = `You are the %s worker executing this plan. Before editing any file, call
the request_ownership tool with that file's path; if denied, pick a
different file or wait and retry. Call release_ownership when done with a
file.

Plan:
%s`

// Implement fans out the fixed Backend/Frontend/Test roles in parallel
// worktrees. All roles are wired to one in-process ownership MCP server
// (over HTTP, sharing one Registry) so real ownership requests are
// mutually exclusive across workers, not just checked in isolation. It
// returns each role's final output, keyed by role name.
func Implement(ctx context.Context, repo, plan, journalPath string) (map[string]string, error) {
	reg, err := loadRegistry(repo)
	if err != nil {
		return nil, fmt.Errorf("implement: %w", err)
	}
	return runImplementRoles(ctx, repo, plan, journalPath, implementRoles, reg)
}

// ImplementScoped re-runs only the given roles against the same on-disk
// registry, for the Validate phase's scoped RPI re-run: re-implement just
// the team that owned a failing path, not the whole pipeline.
func ImplementScoped(ctx context.Context, repo, plan, journalPath string, roles []string) (map[string]string, error) {
	reg, err := loadRegistry(repo)
	if err != nil {
		return nil, fmt.Errorf("implement (scoped): %w", err)
	}
	return runImplementRoles(ctx, repo, plan, journalPath, roles, reg)
}

func loadRegistry(repo string) (*registry.Registry, error) {
	ledgerDir := filepath.Join(repo, ".ledger")
	if err := os.MkdirAll(ledgerDir, 0o755); err != nil {
		return nil, err
	}
	return registry.Load(filepath.Join(ledgerDir, "registry.json"))
}

func runImplementRoles(ctx context.Context, repo, plan, journalPath string, roles []string, reg *registry.Registry) (map[string]string, error) {
	model, err := modelrouting.Choose(ctx, repo, plan)
	if err != nil {
		return nil, fmt.Errorf("implement: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("implement: %w", err)
	}
	server := &http.Server{Handler: ownership.NewHTTPHandler(reg, "/mcp")}
	go server.Serve(listener)
	defer server.Close()
	baseURL := fmt.Sprintf("http://%s/mcp", listener.Addr())

	configDir, err := os.MkdirTemp("", "ledger-mcp-config-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(configDir)

	q := queue.New(settings.LoadDefault().Cap(len(roles)))
	tasks := make(chan queue.Task, len(roles))
	outputs := make(map[string]string, len(roles))
	var outputsMu sync.Mutex

	for _, role := range roles {
		role := role
		branch := "ledger-implement-" + role
		tasks <- queue.Task{
			ID:    role,
			Phase: "implement",
			Run: func(ctx context.Context) error {
				wt, err := worktree.CreateWorktree(repo, branch)
				if err != nil {
					return fmt.Errorf("%s: %w", role, err)
				}
				defer worktree.PruneWorktree(repo, wt, branch)

				configPath := filepath.Join(configDir, role+".json")
				if err := ownership.WriteMCPConfig(configPath, baseURL, role); err != nil {
					return fmt.Errorf("%s: %w", role, err)
				}

				args := append(modelrouting.Args(model),
					"--mcp-config", configPath,
					"--allowed-tools", "mcp__ownership__request_ownership,mcp__ownership__release_ownership")
				out, err := worker.Run(ctx, wt, fmt.Sprintf(implementPromptTmpl, role, plan), args...)
				if err != nil {
					journal.Append(journalPath, "error", map[string]string{"role": role, "error": err.Error()})
					return fmt.Errorf("%s: %w", role, err)
				}
				journal.Append(journalPath, "implement", map[string]string{"role": role, "output": out})

				outputsMu.Lock()
				outputs[role] = out
				outputsMu.Unlock()
				return nil
			},
		}
	}
	close(tasks)

	for id, err := range q.Run(ctx, tasks) {
		if err != nil {
			return nil, fmt.Errorf("implement: %s: %w", id, err)
		}
	}

	journal.Append(journalPath, "phase", map[string]string{"phase": "implement", "status": "done"})
	return outputs, nil
}
