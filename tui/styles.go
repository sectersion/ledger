package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors follow charmbracelet/crush's default "Charmtone Pantera" theme, so
// ledger's TUI reads as part of the same family of tools.
const (
	colorPrimary  = lipgloss.Color("#6B50FF") // Charple
	colorAccent   = lipgloss.Color("#68FFD6") // Bok
	colorFg       = lipgloss.Color("#ECEBF0") // Sash
	colorFgSubtle = lipgloss.Color("#BFBCC8") // Smoke
	colorFgDim    = lipgloss.Color("#605F6B") // Oyster
	colorBg       = lipgloss.Color("#201F26") // Pepper
	colorBgRaised = lipgloss.Color("#2D2C36") // BBQ
	colorBorder   = lipgloss.Color("#3A3943") // Char
	colorSelectBg = lipgloss.Color("#4D4C57") // Iron

	colorRunning = lipgloss.Color("#F5EF34") // Mustard
	colorDone    = lipgloss.Color("#00FFB2") // Julep
	colorFailed  = lipgloss.Color("#FF577D") // Coral
	colorPaused  = lipgloss.Color("#00A4FF") // Malibu
	colorKilled  = lipgloss.Color("#605F6B") // Oyster
)

var (
	barStyle = lipgloss.NewStyle().Bold(true).
			Foreground(colorFg).Background(colorBgRaised).Padding(0, 1)
	dimStyle      = lipgloss.NewStyle().Foreground(colorFgDim)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(colorFg).Background(colorSelectBg)
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Underline(true)

	runningStyle = lipgloss.NewStyle().Foreground(colorRunning)
	doneStyle    = lipgloss.NewStyle().Foreground(colorDone)
	failedStyle  = lipgloss.NewStyle().Foreground(colorFailed)
	pausedStyle  = lipgloss.NewStyle().Foreground(colorPaused)
	killedStyle  = lipgloss.NewStyle().Foreground(colorKilled)

	toastStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Foreground(colorFg).
			Padding(1, 2)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Foreground(colorFg).
			Padding(0, 1)
)

var (
	promptStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	toolUseStyle = lipgloss.NewStyle().Foreground(colorAccent)
)

// styleDetailLine colors a journal-stream line by its leading glyph: the
// initial prompt, a tool call, or a success/failure result each get a
// distinct color instead of one flat foreground.
func styleDetailLine(line string) string {
	switch {
	case strings.HasPrefix(line, "▶ "):
		return promptStyle.Render(line)
	case strings.HasPrefix(line, "→ "):
		return toolUseStyle.Render(line)
	case strings.HasPrefix(line, "✓ "):
		return doneStyle.Render(line)
	case strings.HasPrefix(line, "✗ "):
		return failedStyle.Render(line)
	default:
		return line
	}
}

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
