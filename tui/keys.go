package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/orchestrator"
)

func (m Model) findSelected() *orchestrator.Agent {
	for i := range m.agents {
		if m.agents[i].ID == m.selectedID {
			return &m.agents[i]
		}
	}
	return nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeKillConfirm:
		return m.handleKillConfirmKey(msg)
	case modeGate:
		return m.handleGateKey(msg)
	case modeGateComment:
		return m.handleGateCommentKey(msg)
	case modeBtw:
		return m.handleBtwKey(msg)
	case modeSearch:
		return m.handleSearchKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m Model) handleKillConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.o.Kill(m.selectedID)
		m.setToast("killed " + m.selectedID)
		m.mode = modeNormal
	case "n", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) handleGateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		m.pendingGate.Approve()
		m.setToast("approved " + m.pendingGate.Phase)
		m.pendingGate = nil
		m.mode = modeNormal
	case "r":
		m.pendingGate.Reject("")
		m.setToast("rejected " + m.pendingGate.Phase)
		m.pendingGate = nil
		m.mode = modeNormal
	case "e":
		m.mode = modeGateComment
		m.comment.SetValue("")
		m.comment.Focus()
	case "up", "k":
		m.gateVP.LineUp(1)
	case "down", "j":
		m.gateVP.LineDown(1)
	}
	return m, nil
}

func (m Model) handleGateCommentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		comment := m.comment.Value()
		m.pendingGate.Reject(comment)
		m.setToast("rejected " + m.pendingGate.Phase + ": " + comment)
		m.pendingGate = nil
		m.mode = modeNormal
		return m, nil
	case "esc":
		m.mode = modeGate
		return m, nil
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

// handleBtwKey composes and submits a /btw relay message: queue the
// message for id (consumed by worker.Run via the orchestrator's Relayer on
// the killed run's failure) and kill the selected agent, so its
// respawn — the fan-out loop retrying that role — carries the message
// prepended to its prompt.
func (m Model) handleBtwKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		text := m.btwInput.Value()
		id := m.selectedID
		m.o.Relay(id, text)
		m.o.Kill(id)
		journal.Append(m.journalPath, "btw", map[string]string{"agentID": id, "message": text})
		m.setToast("relayed to " + id + ": " + text)
		m.mode = modeNormal
		return m, nil
	case "esc":
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.btwInput, cmd = m.btwInput.Update(msg)
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.query = m.search.Value()
	if m.cursor >= len(m.rows()) {
		m.cursor = 0
	}
	return m, cmd
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor < len(m.rows())-1 {
			m.cursor++
		}

	case "enter":
		rows := m.rows()
		if m.cursor < len(rows) && rows[m.cursor].isHeader {
			phase := rows[m.cursor].phase
			m.collapsed[phase] = !m.collapsed[phase]
		}

	case " ":
		rows := m.rows()
		if m.cursor < len(rows) && !rows[m.cursor].isHeader {
			m.selectedID = rows[m.cursor].agent.ID
			m.refreshDetail()
		}

	case "k":
		if m.selectedID != "" {
			m.mode = modeKillConfirm
		}

	case "p":
		if a := m.findSelected(); a != nil {
			if a.Status == orchestrator.StatusPaused {
				m.o.Resume(m.selectedID)
				m.setToast("resumed " + m.selectedID)
			} else {
				m.o.Pause(m.selectedID)
				m.setToast("paused " + m.selectedID)
			}
		}

	case "b":
		if m.selectedID != "" {
			m.mode = modeBtw
			m.btwInput.SetValue("")
			m.btwInput.Focus()
		}

	case "/":
		m.mode = modeSearch
		m.search.SetValue(m.query)
		m.search.Focus()
	}
	return m, nil
}
