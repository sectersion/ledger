package tui

import "github.com/charmbracelet/lipgloss"

var (
	barStyle      = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	headerStyle   = lipgloss.NewStyle().Bold(true).Underline(true)

	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	failedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	pausedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	killedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray

	toastStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("11")).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	paneStyle = lipgloss.NewStyle().Padding(0, 1)
)

func statusStyle(s string) lipgloss.Style {
	switch s {
	case "running":
		return runningStyle
	case "done":
		return doneStyle
	case "failed":
		return failedStyle
	case "paused":
		return pausedStyle
	case "killed":
		return killedStyle
	default:
		return dimStyle
	}
}
