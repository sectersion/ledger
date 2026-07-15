// Package tui is the M9 renderer over the orchestrator's live state: a
// collapsible agent tree, a live journal-stream detail pane, review-gate
// and kill-confirm modals, and a /btw relay — all driven by
// orchestrator.Orchestrator, no TUI-only state invented here.
package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/failure"
	"github.com/sectersion/ledger/orchestrator"
	"github.com/sectersion/ledger/settings"
)

type mode int

const (
	modeNormal mode = iota
	modeKillConfirm
	modeGate
	modeGateComment
	modeBtw
	modeSearch
)

// Model is the TUI's Bubble Tea model.
type Model struct {
	o           *orchestrator.Orchestrator
	repo, task  string
	journalPath string

	agents     []orchestrator.Agent
	collapsed  map[string]bool
	cursor     int
	selectedID string
	query      string

	detail   viewport.Model
	gateVP   viewport.Model
	btwInput textinput.Model
	comment  textinput.Model
	search   textinput.Model

	mode        mode
	pendingGate *orchestrator.GateRequest

	phase       string
	toast       string
	toastUntil  time.Time
	finished    bool
	pipelineErr error

	cap int

	width, height int
}

// New builds a Model that, once run, drives RunPipeline against repo for
// task and renders its live state.
func New(o *orchestrator.Orchestrator, repo, task, journalPath string) Model {
	btw := textinput.New()
	btw.Placeholder = "relay message…"
	comment := textinput.New()
	comment.Placeholder = "rejection comment…"
	search := textinput.New()
	search.Placeholder = "search agents…"

	return Model{
		o:           o,
		repo:        repo,
		task:        task,
		journalPath: journalPath,
		collapsed:   map[string]bool{},
		detail:      viewport.New(0, 0),
		gateVP:      viewport.New(0, 0),
		btwInput:    btw,
		comment:     comment,
		search:      search,
		cap:         settings.LoadDefault().ConcurrencyCap,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		startPipeline(m.o, m.repo, m.task, m.journalPath),
		waitForUpdate(m.o.Updates()),
		waitForGate(m.o.Gates()),
		tick(),
	)
}

func (m Model) rows() []row {
	return buildRows(m.agents, m.collapsed, m.query)
}

func (m *Model) setToast(text string) {
	m.toast = text
	m.toastUntil = time.Now().Add(4 * time.Second)
}

func (m *Model) refreshDetail() {
	for _, a := range m.agents {
		if a.ID == m.selectedID {
			m.detail.SetContent(strings.Join(a.Lines, "\n"))
			m.detail.GotoBottom()
			return
		}
	}
	m.detail.SetContent("")
}

func (m *Model) resize() {
	// Top bar + separator + bottom separator + bottom bar.
	paneHeight := m.height - 4
	if paneHeight < 1 {
		paneHeight = 1
	}
	treeWidth := m.width / 3
	detailWidth := m.width - treeWidth - 1
	if detailWidth < 1 {
		detailWidth = 1
	}
	m.detail.Width = detailWidth
	m.detail.Height = paneHeight
	m.gateVP.Width = m.width - 8
	m.gateVP.Height = m.height - 8
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case updateMsg:
		u := orchestrator.Update(msg)
		if u.AgentID == "" && u.Phase != "" {
			m.phase = u.Phase
			m.setToast(fmt.Sprintf("phase: %s", u.Phase))
		}
		m.agents = m.o.Snapshot()
		m.refreshDetail()
		return m, waitForUpdate(m.o.Updates())

	case gateMsg:
		m.pendingGate = msg.req
		m.mode = modeGate
		m.gateVP.SetContent(msg.req.Artifact)
		m.gateVP.GotoTop()
		return m, waitForGate(m.o.Gates())

	case pipelineDoneMsg:
		m.finished = true
		m.pipelineErr = msg.err
		if msg.err != nil {
			var esc *failure.Escalation
			if errors.As(msg.err, &esc) {
				m.setToast(fmt.Sprintf("escalated: %s", esc.Error()))
			} else {
				m.setToast("pipeline failed: " + msg.err.Error())
			}
		} else {
			m.setToast("pipeline done")
		}
		return m, nil

	case tickMsg:
		if !m.toastUntil.IsZero() && time.Now().After(m.toastUntil) {
			m.toast = ""
		}
		return m, tick()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}
