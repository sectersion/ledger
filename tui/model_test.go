package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/orchestrator"
)

func testModel(t *testing.T) Model {
	t.Helper()
	journalPath := filepath.Join(t.TempDir(), "orchestrator.jsonl")
	m := New(orchestrator.New(), t.TempDir(), "task", journalPath)
	m.width, m.height = 100, 30
	m.resize()
	m.agents = []orchestrator.Agent{
		{ID: "ledger-research-0", Phase: "research", Status: orchestrator.StatusDone, Lines: []string{"line one", "line two"}},
		{ID: "ledger-research-1", Phase: "research", Status: orchestrator.StatusRunning},
	}
	return m
}

func key(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
func runes(s string) tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestCursorMovementBounds(t *testing.T) {
	m := testModel(t)
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	mi, _ := m.Update(key(tea.KeyUp))
	m = mi.(Model)
	if m.cursor != 0 {
		t.Fatalf("cursor went below 0: %d", m.cursor)
	}

	for i := 0; i < 10; i++ {
		mi, _ = m.Update(key(tea.KeyDown))
		m = mi.(Model)
	}
	maxRow := len(m.rows()) - 1
	if m.cursor != maxRow {
		t.Fatalf("cursor = %d, want clamped to %d", m.cursor, maxRow)
	}
}

func TestSpaceSelectsAgentUnderCursor(t *testing.T) {
	m := testModel(t)
	// row 0 = "research" header, row 1 = ledger-research-0.
	mi, _ := m.Update(key(tea.KeyDown))
	m = mi.(Model)
	mi, _ = m.Update(key(tea.KeySpace))
	m = mi.(Model)

	if m.selectedID != "ledger-research-0" {
		t.Fatalf("selectedID = %q, want ledger-research-0", m.selectedID)
	}
	if m.detail.View() == "" {
		t.Fatal("expected detail viewport to show the selected agent's lines")
	}
}

func TestKillConfirmFlow(t *testing.T) {
	m := testModel(t)
	m.selectedID = "ledger-research-0"

	mi, _ := m.Update(runes("k"))
	m = mi.(Model)
	if m.mode != modeKillConfirm {
		t.Fatalf("mode = %v, want modeKillConfirm", m.mode)
	}

	mi, _ = m.Update(runes("n"))
	m = mi.(Model)
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want modeNormal after 'n'", m.mode)
	}

	mi, _ = m.Update(runes("k"))
	m = mi.(Model)
	mi, _ = m.Update(runes("y"))
	m = mi.(Model)
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want modeNormal after 'y'", m.mode)
	}
}

func TestSearchFiltersRows(t *testing.T) {
	m := testModel(t)
	mi, _ := m.Update(runes("/"))
	m = mi.(Model)
	if m.mode != modeSearch {
		t.Fatalf("mode = %v, want modeSearch", m.mode)
	}

	mi, _ = m.Update(runes("research-0"))
	m = mi.(Model)
	if m.query != "research-0" {
		t.Fatalf("query = %q, want research-0", m.query)
	}

	rows := m.rows()
	if len(rows) != 2 { // header + the one matching agent
		t.Fatalf("got %d rows, want 2: %+v", len(rows), rows)
	}

	mi, _ = m.Update(key(tea.KeyEsc))
	m = mi.(Model)
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want modeNormal after esc", m.mode)
	}
}

func TestBtwFlowKillsAndJournalsRelay(t *testing.T) {
	m := testModel(t)
	m.selectedID = "ledger-research-0"

	mi, _ := m.Update(runes("b"))
	m = mi.(Model)
	if m.mode != modeBtw {
		t.Fatalf("mode = %v, want modeBtw", m.mode)
	}

	mi, _ = m.Update(runes("do X first"))
	m = mi.(Model)
	mi, _ = m.Update(key(tea.KeyEnter))
	m = mi.(Model)

	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want modeNormal after submit", m.mode)
	}

	entries, err := journal.Read(m.journalPath)
	if err != nil {
		t.Fatalf("journal.Read: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Kind == "btw" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a btw journal entry after submitting the relay")
	}
}

func TestViewRendersWithoutPanicAcrossModes(t *testing.T) {
	for _, mode := range []mode{modeNormal, modeKillConfirm, modeBtw, modeSearch} {
		m := testModel(t)
		m.selectedID = "ledger-research-0"
		m.mode = mode
		if out := m.View(); out == "" {
			t.Fatalf("mode %v: View() returned empty output", mode)
		}
	}
}
