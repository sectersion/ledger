package tui

import (
	"testing"

	"github.com/sectersion/ledger/orchestrator"
)

func TestBuildRowsGroupsByPipelineOrder(t *testing.T) {
	agents := []orchestrator.Agent{
		{ID: "ledger-implement-Backend", Phase: "implement", Status: orchestrator.StatusRunning},
		{ID: "ledger-research-1", Phase: "research", Status: orchestrator.StatusDone},
		{ID: "ledger-research-0", Phase: "research", Status: orchestrator.StatusDone},
	}
	rows := buildRows(agents, nil, "")

	// research header, research-0, research-1, implement header, implement-Backend
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5: %+v", len(rows), rows)
	}
	if !rows[0].isHeader || rows[0].phase != "research" {
		t.Fatalf("row 0 = %+v, want research header", rows[0])
	}
	if rows[1].agent.ID != "ledger-research-0" || rows[2].agent.ID != "ledger-research-1" {
		t.Fatalf("research agents not sorted: %+v, %+v", rows[1], rows[2])
	}
	if !rows[3].isHeader || rows[3].phase != "implement" {
		t.Fatalf("row 3 = %+v, want implement header", rows[3])
	}
}

func TestBuildRowsCollapsesPhase(t *testing.T) {
	agents := []orchestrator.Agent{
		{ID: "ledger-research-0", Phase: "research"},
	}
	rows := buildRows(agents, map[string]bool{"research": true}, "")
	if len(rows) != 1 || !rows[0].isHeader {
		t.Fatalf("expected only the header row when collapsed, got %+v", rows)
	}
}

func TestBuildRowsFiltersBySearchQuery(t *testing.T) {
	agents := []orchestrator.Agent{
		{ID: "ledger-implement-Backend", Phase: "implement"},
		{ID: "ledger-implement-Frontend", Phase: "implement"},
	}
	rows := buildRows(agents, nil, "back")
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (header + Backend only): %+v", len(rows), rows)
	}
	if rows[1].agent.ID != "ledger-implement-Backend" {
		t.Fatalf("expected Backend to match, got %+v", rows[1])
	}
}

func TestBuildRowsDropsPhaseWithNoMatches(t *testing.T) {
	agents := []orchestrator.Agent{
		{ID: "ledger-research-0", Phase: "research"},
	}
	rows := buildRows(agents, nil, "nomatch")
	if len(rows) != 0 {
		t.Fatalf("expected no rows for a query matching nothing, got %+v", rows)
	}
}
