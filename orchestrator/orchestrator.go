// Package orchestrator drives the full pipeline (research -> plan ->
// implement -> validate -> review -> ship) and exposes its live state —
// per-agent status/journal lines, phase transitions, and human review
// gates — for a renderer (the M9 TUI) to observe and act on. No
// TUI-only state is invented here: everything is derived from the
// worker/queue/journal/registry/phases/failure primitives M1-M8 already
// built.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sectersion/ledger/failure"
	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/phases"
	"github.com/sectersion/ledger/registry"
	"github.com/sectersion/ledger/worker"
)

// Status is an agent's lifecycle state.
type Status string

const (
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
	StatusPaused  Status = "paused"
	StatusKilled  Status = "killed"
)

// Agent is one worker the orchestrator has seen, keyed by its agent ID
// (worker.WithAgentID, or the worktree's basename by default).
type Agent struct {
	ID     string
	Phase  string
	Status Status
	Lines  []string
}

// Update is an incremental change: either to a specific agent (AgentID
// set), or a pipeline-wide phase transition (AgentID empty).
type Update struct {
	AgentID string
	Phase   string
	Status  Status
	Line    string
}

// GateRequest is a blocking human review gate (after Research, Plan, and
// Review sign-off). RunPipeline blocks on it until Approve/Reject is
// called — from the TUI, in response to a/r keypresses.
type GateRequest struct {
	Phase    string
	Artifact string
	resp     chan gateDecision
}

type gateDecision struct {
	approved bool
	comment  string
}

// Approve unblocks the pipeline to continue past this gate.
func (g *GateRequest) Approve() { g.resp <- gateDecision{approved: true} }

// Reject unblocks the pipeline to stop, recording comment as the reason.
func (g *GateRequest) Reject(comment string) {
	g.resp <- gateDecision{approved: false, comment: comment}
}

// Orchestrator runs one pipeline invocation and publishes its live state.
type Orchestrator struct {
	mu           sync.Mutex
	agents       map[string]*Agent
	order        []string
	cancels      map[string]context.CancelFunc
	relays       map[string]string
	currentPhase string
	journalPath  string

	updates chan Update
	gates   chan *GateRequest
}

// New creates an Orchestrator. Updates/Gates must be drained by the
// caller (typically the TUI) for RunPipeline to make progress past a
// gate; Updates is best-effort (buffered, drops if full — a stale UI can
// always call Snapshot to resync).
func New() *Orchestrator {
	return &Orchestrator{
		agents:  map[string]*Agent{},
		cancels: map[string]context.CancelFunc{},
		relays:  map[string]string{},
		updates: make(chan Update, 256),
		gates:   make(chan *GateRequest),
	}
}

// Updates is the live event stream: per-agent status/lines and
// pipeline-wide phase transitions.
func (o *Orchestrator) Updates() <-chan Update { return o.updates }

// Gates is the stream of pending human review gates.
func (o *Orchestrator) Gates() <-chan *GateRequest { return o.gates }

// Snapshot returns every agent seen so far, in first-seen order.
func (o *Orchestrator) Snapshot() []Agent {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]Agent, 0, len(o.order))
	for _, id := range o.order {
		out = append(out, *o.agents[id])
	}
	return out
}

// Phase returns the pipeline's current top-level phase.
func (o *Orchestrator) Phase() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.currentPhase
}

// Kill cancels the named agent's worker (and its context) immediately —
// destructive and irreversible, per PLAN.md's TUI spec (the confirm
// modal lives in the TUI layer, not here).
func (o *Orchestrator) Kill(id string) {
	o.mu.Lock()
	cancel := o.cancels[id]
	if a, ok := o.agents[id]; ok {
		a.Status = StatusKilled
	}
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Relay queues message to be prepended to id's prompt the next time its
// role respawns — paired with Kill, this is `/btw`'s actual effect: kill
// the running worker, then relay the message into its replacement.
func (o *Orchestrator) Relay(id, message string) {
	o.mu.Lock()
	o.relays[id] = message
	o.mu.Unlock()
}

// TakeRelay returns and clears any message queued by Relay for id. It's
// worker.Relayer, consulted by worker.Run after a failed run.
func (o *Orchestrator) TakeRelay(id string) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	msg, ok := o.relays[id]
	if ok {
		delete(o.relays, id)
	}
	return msg, ok
}

// Pause marks an agent paused and Resume marks it running again.
//
// ponytail: this is a logical/UI-visible pause only, not a real OS
// process suspend (no portable SIGSTOP equivalent on Windows via
// os/exec) — the worker keeps running underneath. Upgrade to a real
// pause if that gap ever matters: platform-specific process suspend, or
// have the worker poll a pause flag between tool calls.
func (o *Orchestrator) Pause(id string)  { o.setStatus(id, StatusPaused) }
func (o *Orchestrator) Resume(id string) { o.setStatus(id, StatusRunning) }

func (o *Orchestrator) setStatus(id string, status Status) {
	o.mu.Lock()
	if a, ok := o.agents[id]; ok {
		a.Status = status
	}
	o.mu.Unlock()
	o.publish(Update{AgentID: id, Status: status})
}

// sink is worker.Sink: every event any worker (anywhere in the pipeline)
// emits is turned into an Update.
func (o *Orchestrator) sink(agentID string, e worker.Event) {
	status := StatusRunning
	if e.Type == "result" {
		var r struct {
			IsError bool `json:"is_error"`
		}
		if json.Unmarshal(e.Raw, &r) == nil {
			if r.IsError {
				status = StatusFailed
			} else {
				status = StatusDone
			}
		}
	}
	o.emit(Update{AgentID: agentID, Status: status, Line: e.Summary()})
}

// registerCancel is worker.Registrar: every worker registers its own
// cancel func, keyed by agent ID, for Kill.
func (o *Orchestrator) registerCancel(id string, cancel context.CancelFunc) {
	o.mu.Lock()
	o.cancels[id] = cancel
	o.mu.Unlock()
}

func (o *Orchestrator) emit(u Update) {
	o.mu.Lock()
	a, ok := o.agents[u.AgentID]
	if !ok {
		a = &Agent{ID: u.AgentID, Phase: o.currentPhase}
		o.agents[u.AgentID] = a
		o.order = append(o.order, u.AgentID)
	}
	if u.Status != "" {
		a.Status = u.Status
	}
	if u.Line != "" {
		a.Lines = append(a.Lines, u.Line)
	}
	u.Phase = a.Phase
	o.mu.Unlock()
	o.publish(u)
}

func (o *Orchestrator) publish(u Update) {
	select {
	case o.updates <- u:
	default:
	}
}

func (o *Orchestrator) setPhase(phase string) {
	o.mu.Lock()
	o.currentPhase = phase
	o.mu.Unlock()
	o.publish(Update{Phase: phase})
	journal.Append(o.journalPath, "phase", map[string]string{"phase": phase, "status": "started"})
}

// requestGate blocks until the TUI (or any Gates() consumer) approves or
// rejects.
func (o *Orchestrator) requestGate(phase, artifact string) (bool, string) {
	g := &GateRequest{Phase: phase, Artifact: artifact, resp: make(chan gateDecision, 1)}
	o.gates <- g
	d := <-g.resp
	return d.approved, d.comment
}

// RunPipeline drives all 6 phases against repo for task, blocking on the
// Research/Plan/Review gates and scoping a re-run to just the
// failing-path owners on a Validate failure, per PLAN.md.
func (o *Orchestrator) RunPipeline(ctx context.Context, repo, task, journalPath string) error {
	o.journalPath = journalPath
	ctx = worker.WithSink(ctx, o.sink)
	ctx = worker.WithRegistrar(ctx, o.registerCancel)
	ctx = worker.WithRelayer(ctx, o.TakeRelay)

	o.setPhase("research")
	reportPath, err := phases.Research(ctx, repo, task, journalPath)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	report, err := readFileString(reportPath)
	if err != nil {
		return err
	}
	if approved, comment := o.requestGate("research", report); !approved {
		return fmt.Errorf("research gate rejected: %s", comment)
	}

	o.setPhase("plan")
	planPath, err := phases.Plan(ctx, repo, report, journalPath)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	plan, err := readFileString(planPath)
	if err != nil {
		return err
	}
	if approved, comment := o.requestGate("plan", plan); !approved {
		return fmt.Errorf("plan gate rejected: %s", comment)
	}

	o.setPhase("implement")
	if _, err := phases.Implement(ctx, repo, plan, journalPath); err != nil {
		return fmt.Errorf("implement: %w", err)
	}

	o.setPhase("validate")
	result, err := phases.Validate(ctx, repo, plan, journalPath)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	for attempt := 1; !result.Passed && attempt <= failure.DefaultRetries; attempt++ {
		reg, err := registry.Load(filepath.Join(repo, ".ledger", "registry.json"))
		if err != nil {
			return fmt.Errorf("validate: %w", err)
		}
		owners := phases.FailingOwners(reg, result.FailedPackages)
		if len(owners) == 0 {
			break // nothing to scope a re-run to; fall through to escalation below
		}
		if _, err := phases.ImplementScoped(ctx, repo, plan, journalPath, owners); err != nil {
			return fmt.Errorf("validate: scoped re-run: %w", err)
		}
		if result, err = phases.Validate(ctx, repo, plan, journalPath); err != nil {
			return fmt.Errorf("validate: %w", err)
		}
	}
	if !result.Passed {
		return &failure.Escalation{Phase: "validate", Attempts: failure.DefaultRetries + 1, Cause: fmt.Errorf("still failing after scoped re-runs: %v", result.FailedPackages)}
	}

	o.setPhase("review")
	review, err := phases.Review(ctx, repo, plan, journalPath)
	if err != nil {
		return fmt.Errorf("review: %w", err)
	}
	approved, comment := o.requestGate("review", review.Report)
	if err := phases.Approve(journalPath, approved); err != nil {
		return err
	}
	if !approved {
		return fmt.Errorf("review gate rejected: %s", comment)
	}

	o.setPhase("ship")
	if _, err := phases.Ship(ctx, repo, journalPath); err != nil {
		return fmt.Errorf("ship: %w", err)
	}

	o.setPhase("done")
	return nil
}

func readFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
