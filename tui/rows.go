package tui

import (
	"sort"
	"strings"

	"github.com/sectersion/ledger/orchestrator"
)

// row is one line of the collapsible agent tree: either a phase header
// (agentID empty) or an agent under it.
type row struct {
	isHeader bool
	phase    string
	agent    orchestrator.Agent
}

// phaseOrder is the fixed pipeline order, so the tree groups phases in
// the order they run rather than alphabetically/insertion order (which
// can interleave once concurrent agents from different phases overlap).
var phaseOrder = []string{"research", "plan", "implement", "validate", "review", "ship"}

func phaseRank(phase string) int {
	for i, p := range phaseOrder {
		if p == phase {
			return i
		}
	}
	return len(phaseOrder)
}

// buildRows groups agents by phase (fixed pipeline order), sorts agents
// within a phase by ID for a stable display, skips phases collapsed in
// collapsed, and filters agent rows by a case-insensitive substring match
// on agent ID/name when query is non-empty (PLAN.md's `/` search: filters
// the tree by agent name/id only).
func buildRows(agents []orchestrator.Agent, collapsed map[string]bool, query string) []row {
	byPhase := map[string][]orchestrator.Agent{}
	var phases []string
	seen := map[string]bool{}
	for _, a := range agents {
		if !seen[a.Phase] {
			seen[a.Phase] = true
			phases = append(phases, a.Phase)
		}
		byPhase[a.Phase] = append(byPhase[a.Phase], a)
	}
	sort.Slice(phases, func(i, j int) bool { return phaseRank(phases[i]) < phaseRank(phases[j]) })

	query = strings.ToLower(strings.TrimSpace(query))

	var rows []row
	for _, phase := range phases {
		agentsInPhase := byPhase[phase]
		sort.Slice(agentsInPhase, func(i, j int) bool { return agentsInPhase[i].ID < agentsInPhase[j].ID })

		var matched []orchestrator.Agent
		for _, a := range agentsInPhase {
			if query == "" || strings.Contains(strings.ToLower(a.ID), query) {
				matched = append(matched, a)
			}
		}
		if len(matched) == 0 {
			continue
		}

		rows = append(rows, row{isHeader: true, phase: phase})
		if collapsed[phase] {
			continue
		}
		for _, a := range matched {
			rows = append(rows, row{phase: phase, agent: a})
		}
	}
	return rows
}

// statusGlyph is the short marker shown next to an agent row.
func statusGlyph(s orchestrator.Status) string {
	switch s {
	case orchestrator.StatusRunning:
		return "●"
	case orchestrator.StatusDone:
		return "✓"
	case orchestrator.StatusFailed:
		return "✗"
	case orchestrator.StatusPaused:
		return "⏸"
	case orchestrator.StatusKilled:
		return "☠"
	default:
		return "⏳"
	}
}
