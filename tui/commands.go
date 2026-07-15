package tui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/orchestrator"
)

// updateMsg is one orchestrator.Update delivered to the TUI.
type updateMsg orchestrator.Update

// gateMsg is a pending human review gate.
type gateMsg struct{ req *orchestrator.GateRequest }

// pipelineDoneMsg carries RunPipeline's final result.
type pipelineDoneMsg struct{ err error }

// tickMsg drives toast expiry.
type tickMsg time.Time

// gitTickMsg paces the sidebar's modified-files poll.
type gitTickMsg time.Time

// gitStatusMsg carries the sidebar's modified-files list.
type gitStatusMsg struct{ files []string }

func waitForUpdate(ch <-chan orchestrator.Update) tea.Cmd {
	return func() tea.Msg {
		u, ok := <-ch
		if !ok {
			return nil
		}
		return updateMsg(u)
	}
}

func waitForGate(ch <-chan *orchestrator.GateRequest) tea.Cmd {
	return func() tea.Msg {
		g, ok := <-ch
		if !ok {
			return nil
		}
		return gateMsg{req: g}
	}
}

func startPipeline(o *orchestrator.Orchestrator, repo, task, journalPath string) tea.Cmd {
	return func() tea.Msg {
		err := o.RunPipeline(context.Background(), repo, task, journalPath)
		return pipelineDoneMsg{err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func gitTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return gitTickMsg(t) })
}

// fetchGitStatus lists modified/untracked paths in repo for the sidebar.
// Best-effort: any error (not a git repo, git missing) just yields no
// files rather than failing the TUI.
func fetchGitStatus(repo string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
		if err != nil {
			return gitStatusMsg{}
		}
		var files []string
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			if len(line) > 3 {
				files = append(files, strings.TrimSpace(line[3:]))
			}
		}
		return gitStatusMsg{files: files}
	}
}
