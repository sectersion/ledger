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

	body := lipgloss.JoinHorizontal(lipgloss.Top, m.treeView(), m.detailView())
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
	width := m.width/3 - 2
	if width < 1 {
		width = 1
	}
	return paneStyle.Width(width).Height(m.height - 4).Render(b.String())
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
