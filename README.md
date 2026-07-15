# ledger

Point `ledger` at a repo and a task description, and it runs a headless team
of Claude Code agents through a fixed research → plan → implement → validate
→ review → ship pipeline — each stage isolated in its own git worktree —
while you watch and approve in a terminal UI.

Think of it as a project manager for large, multi-file changes (security
audits, framework migrations, full-codebase overhauls) that would otherwise
mean babysitting one long Claude Code session: it splits the work across
parallel subagents, keeps them from stepping on each other's files, and
stops for your sign-off at the points that matter.

## Why

A single Claude Code session handling a big task tends to lose context,
overwrite its own earlier work, or need constant hand-holding. `ledger`
fixes that by:

- **Splitting work into roles** — research, planning, and implementation
  each fan out to specialized subagents (e.g. Codebase Locator, Risk
  Analyzer, Backend/Frontend/Test workers) instead of one agent doing
  everything serially.
- **Isolating every agent in its own git worktree** so parallel work can't
  collide, with a shared ownership registry so two agents never write the
  same file at once.
- **Gating on human review** after research and planning, and before
  shipping — nothing merges without your approval.
- **Recovering from failure automatically** — a worker that fails gets
  killed and respawned (up to a retry cap) rather than leaving the
  pipeline stuck.

## Install

```
go build ./...
```

Requires the `claude` CLI on your `PATH` (ledger spawns it as a subprocess —
no SDK, no API key management of its own).

## Usage

```
go run ./cmd/ledger run <repo> "<task description>"   # run the pipeline + TUI
go run ./cmd/ledger prune [repo]                       # remove leftover worktrees
```

This opens a split-pane terminal UI: an agent tree on one side, a live
journal stream on the other.

**Keybindings:** arrows to move the cursor, `space` to select an agent,
`enter` to collapse/expand a phase, `k` to kill an agent (with confirmation),
`p` to pause/resume, `b` for `/btw` (kill an agent and relay a message to
its replacement), `/` to search agents by name or id.

**Review gates** (after Research, Plan, and before Ship) pop up as a modal:
`a` to approve, `r` to reject, `e` to edit-and-comment. The pipeline halts
until you respond.

Concurrency cap and allowed models are configured in
`~/.ledger/settings.json`.

## Demo

See [DEMO.md](DEMO.md) for a recording (or the steps to make your own) of
the pipeline running end to end against a small repo — parallel worker
roles, a review gate firing, and the final shipped branch.

## Why a fixed pipeline

Tools like SWE-agent or `aider --architect` give the model a freeform
loop — plan, act, observe, repeat — and let it decide what to do next at
every step. That's more flexible, but it also means the model can wander:
skip validation, re-plan mid-implementation, or silently touch a file
another instance is also editing.

`ledger` picks the opposite tradeoff: research → plan → implement →
validate → review → ship is fixed and always runs in that order. You give
up the ability for the agent to decide "actually, skip validation this
time" — but in exchange you get a pipeline that's predictable to review,
where every stage produces an artifact you can gate on, and where a stuck
or wrong agent fails one bounded stage instead of the whole run. For big,
multi-file changes where the failure mode you're avoiding is "confidently
wrong across 20 files," predictability wins. For open-ended exploration or
single-file fixes, a freeform loop is faster and the fixed structure is
just overhead.

## Comparison

|                       | ledger                          | SWE-agent                     | `aider --architect`         | Devin                          |
|-----------------------|----------------------------------|--------------------------------|-------------------------------|--------------------------------|
| Pipeline structure    | Fixed 6-stage (research→plan→implement→validate→review→ship) | Freeform ReAct-style agent loop | Freeform: architect model plans, editor model acts, loop | Freeform, autonomous planning loop |
| Isolation model       | One git worktree per role/agent, shared ownership registry (mutually exclusive file locks) | Single sandboxed container, one agent | Single working tree, no multi-agent isolation | Own cloud VM/sandbox per session |
| Human-in-the-loop     | Blocking gates after Research, Plan, and before Ship (approve/reject/edit) | Optional human feedback, not staged | Interactive per-edit confirmation | Async check-ins, not staged gates |
| Self-hosting          | Yes — plain Go binary + your own `claude` CLI, no hosted service | Yes — open source, run locally or in CI | Yes — open source, local | No — hosted product only |

This is a self-assessment, not a benchmark — the tools solve overlapping
but not identical problems (SWE-agent targets issue-resolution
benchmarks, Devin targets a hosted autonomous-engineer product). Treat it
as "what tradeoff did each one pick," not a leaderboard.

## How it works

1. **Research** — parallel subagents (Codebase Locator/Analyzer, Pattern
   Finder, External Research, Historical Context) explore the repo and
   compress their findings into one report. → review gate.
2. **Plan** — Architecture Planner, Risk Analyzer, and Test Planner turn
   the research into a concrete spec and rubric. → review gate.
3. **Implement** — Backend/Frontend/Test workers execute the plan, each in
   its own worktree, requesting file ownership from a shared registry as
   they go so they never overlap.
4. **Validate** — runs `go vet`/`go test`/plan-compliance checks. A
   failure triggers a scoped re-run of just the failing slice, not the
   whole pipeline.
5. **Code Review** — an automated security/quality pass, pending your
   sign-off.
6. **Commit and Ship** — opens a PR via `gh pr create`, falling back to a
   plain `git push`/branch handoff if that's unavailable.

## Status

M0–M10 are implemented, including model routing and the Bubble Tea TUI,
with tests that exercise the real `claude` CLI end to end (skipped
automatically if `claude` isn't on `PATH`).

Known gaps, deliberately scoped rather than silently missing:
- Pause/Resume are UI-visible only, not a real OS process suspend (no
  portable SIGSTOP equivalent via `os/exec` on Windows).
- The standalone stdio ownership MCP server (`cmd/ledger-ownership-mcp`)
  isn't used by `Implement` anymore — it shares one in-process registry
  over HTTP instead to avoid multi-process write races — but is kept
  around as a dev tool.

**Out of scope for v1:** multi-orchestrator coordination, a
user-configurable pipeline shape, live registry editing via the TUI, and
detach/reattach beyond what the OS's own job control gives you.

## For contributors

See [PLAN.md](PLAN.md) for the full design and
[IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for how it was built,
milestone by milestone.

```
cmd/ledger/                orchestrator + TUI entrypoint (`run`, `prune`)
cmd/ledger-ownership-mcp/  standalone stdio ownership MCP server (dev tool)
worker/                    spawn a headless claude CLI worker, stream stream-json events
worktree/                  git worktree add/remove/list wrappers
journal/                   per-agent append-only JSONL journal
registry/                  path -> owning agentID lock registry
queue/                     in-memory FIFO task queue with a concurrency cap
ownership/                 ownership MCP server (request/release_ownership)
failure/                   kill-and-respawn, retry cap + escalation, /btw primitive
modelrouting/               one-off "best model for this job" routing
settings/                  ~/.ledger/settings.json loader
phases/                    research, plan, implement, validate, review, ship
orchestrator/              wires the 6 phases together, live agent/gate state
tui/                       Bubble Tea split-pane UI (agent tree + journal stream)
```
