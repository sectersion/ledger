package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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
	if _, err := tea.NewProgram(model{}, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
