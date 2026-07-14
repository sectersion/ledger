package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/worktree"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "q" || k.String() == "ctrl+c") {
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string { return "ledger\n\npress q to quit\n" }

func main() {
	if len(os.Args) > 1 && os.Args[1] == "prune" {
		runPrune()
		return
	}

	if _, err := tea.NewProgram(model{}, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPrune() {
	repo, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := worktree.PruneAll(repo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
