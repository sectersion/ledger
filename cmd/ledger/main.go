package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/orchestrator"
	"github.com/sectersion/ledger/tui"
	"github.com/sectersion/ledger/worktree"
)

const usage = `usage:
  ledger run <repo> <task description>
  ledger prune [repo]`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runPipeline(os.Args[2:])
	case "prune":
		runPrune(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

func runPipeline(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	repo, task := args[0], args[1]

	journalPath := filepath.Join(repo, ".ledger", "orchestrator.jsonl")
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	o := orchestrator.New()
	m := tui.New(o, repo, task, journalPath)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPrune(args []string) {
	repo := "."
	if len(args) > 0 {
		repo = args[0]
	}
	repo, err := filepath.Abs(repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := worktree.PruneAll(repo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
