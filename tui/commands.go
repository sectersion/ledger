package tui

import (
	"context"
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
