package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "starting…"
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar(), m.treeView(), m.detailView())
	page := lipgloss.JoinVertical(lipgloss.Left, m.topBar(), body, m.bottomBar())

	switch m.mode {
	case modeKillConfirm:
		return m.overlay(page, m.killConfirmModal())
	case modeGate:
		return m.overlay(page, m.gateModal())
	case modeGateComment:
		return m.overlay(page, m.gateCommentModal())
	case modeBtw:
		return m.overlay(page, m.btwModal())
	default:
		return page
	}
}

func (m Model) topBar() string {
	sub := "…"
	if a := m.findSelected(); a != nil && len(a.Lines) > 0 {
		sub = a.Lines[len(a.Lines)-1]
	}
	text := fmt.Sprintf("ledger · stage: %s · substage: %s", orDash(m.phase), truncate(sub, 60))
	return barStyle.Width(m.width).Render(text)
}

func (m Model) bottomBar() string {
	running := 0
	for _, a := range m.agents {
		if a.Status == "running" {
			running++
		}
	}
	gate := "none pending"
	if m.pendingGate != nil {
		gate = m.pendingGate.Phase + " awaiting sign-off"
	}
	keys := "[k]ill [p]ause [b]tw [/]search"
	if m.mode == modeSearch {
		keys = "search: " + m.search.View() + "  [enter/esc] done"
	}
	text := fmt.Sprintf("workers: %d/%d · gate: %s · %s", running, m.cap, gate, keys)
	line := barStyle.Width(m.width).Render(text)
	if m.toast != "" {
		return line + "\n" + toastStyle.Render(m.toast)
	}
	return line
}

func (m Model) treeView() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("AGENTS"))
	b.WriteString("\n")
	for i, r := range m.rows() {
		var line string
		if r.isHeader {
			marker := "▾"
			if m.collapsed[r.phase] {
				marker = "▸"
			}
			line = fmt.Sprintf("%s %s", marker, r.phase)
			line = headerStyle.Render(line)
		} else {
			marker := "  "
			if r.agent.ID == m.selectedID {
				marker = "▸ "
			}
			glyph := statusStyle(string(r.agent.Status)).Render(statusGlyph(r.agent.Status))
			line = fmt.Sprintf("%s%s %s", marker, glyph, r.agent.ID)
		}
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	avail := m.width - m.sidebarWidth()
	if avail < 0 {
		avail = 0
	}
	width := avail/3 - 4
	if width < 1 {
		width = 1
	}
	return paneStyle.Width(width).Height(m.height - 6).Render(b.String())
}

// minWidthForLogo is the terminal width below which the block-letter logo
// (35 cols) plus the tree/detail panes it leaves room for would clip — a
// plain text wordmark is used instead.
const minWidthForLogo = 100

// sidebarContentWidth returns the sidebar's content width, shrunk to fit
// narrow terminals instead of always reserving room for the full-size
// block logo.
func (m Model) sidebarContentWidth() int {
	width := sidebarContentWidth
	if m.width < minWidthForLogo {
		width = m.width / 4
		if width < 12 {
			width = 12
		}
	}
	if width > m.width {
		width = m.width
	}
	return width
}

// sidebarWidth is the sidebar's total rendered width, content plus
// paneStyle's border and padding.
func (m Model) sidebarWidth() int {
	return m.sidebarContentWidth() + 4
}

// sidebar shows the "LEDGER" wordmark (gradient block art if there's room,
// a plain styled word otherwise) and the repo's currently modified/
// untracked paths (polled via git status, see gitTick).
func (m Model) sidebar() string {
	width := m.sidebarContentWidth()

	var b strings.Builder
	if m.width < minWidthForLogo {
		b.WriteString(headerStyle.Render("LEDGER"))
	} else {
		b.WriteString(renderLogo())
	}
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("MODIFIED"))
	b.WriteString("\n")
	if len(m.modifiedFiles) == 0 {
		b.WriteString(dimStyle.Render("clean"))
	} else {
		for _, f := range m.modifiedFiles {
			b.WriteString(dimStyle.Render(truncate(f, width)))
			b.WriteString("\n")
		}
	}
	return paneStyle.Width(width).Height(m.height - 6).Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) detailView() string {
	title := "agent: " + orDash(m.selectedID)
	return paneStyle.Render(headerStyle.Render(title) + "\n" + m.detail.View())
}

func (m Model) overlay(background, modal string) string {
	return background + "\n" + lipgloss.Place(m.width, 3, lipgloss.Center, lipgloss.Top, modal)
}

func (m Model) killConfirmModal() string {
	return modalStyle.Render(fmt.Sprintf("kill %s? y/n", m.selectedID))
}

func (m Model) gateModal() string {
	title := fmt.Sprintf("review gate: %s", m.pendingGate.Phase)
	body := m.gateVP.View()
	footer := "[a]pprove [r]eject [e]dit-comment · j/k/arrows scroll"
	return modalStyle.Render(headerStyle.Render(title) + "\n\n" + body + "\n\n" + dimStyle.Render(footer))
}

func (m Model) gateCommentModal() string {
	title := fmt.Sprintf("reject %s: comment", m.pendingGate.Phase)
	return modalStyle.Render(headerStyle.Render(title) + "\n\n" + m.comment.View() + "\n\n" + dimStyle.Render("[enter] submit  [esc] back"))
}

func (m Model) btwModal() string {
	title := "/btw " + m.selectedID
	return modalStyle.Render(headerStyle.Render(title) + "\n\n" + m.btwInput.View() + "\n\n" + dimStyle.Render("[enter] send (kills & relays)  [esc] cancel"))
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
