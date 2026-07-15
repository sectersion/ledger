# ledger

An orchestration layer outside Claude Code. A single orchestrator process
drives headless Claude Code instances as workers to execute large tasks
(security audits, framework migrations, full-codebase overhauls) through a
fixed 6-phase pipeline, with git worktree isolation, per-agent journals, and
human review gates.

See [PLAN.md](PLAN.md) for the full design and [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)
for how it was built, milestone by milestone.

## Stack

Go, Bubble Tea (TUI), the `claude` CLI spawned as a subprocess (no SDK),
plain JSON/JSONL files for state (no DB).

## Pipeline

1. **Research** — fixed subagent roles (Codebase Locator/Analyzer, Pattern
   Finder, External Research, Historical Context) fan out in parallel
   worktrees, aggregate into a compressed report. Blocking human review gate.
2. **Plan** — Architecture Planner/Risk Analyzer/Test Planner roles produce
   a spec + rubric. Blocking human review gate.
3. **Implement** — Backend/Frontend/Test workers execute the plan, each in
   its own git worktree, requesting file ownership from one shared registry
   (served over an in-process HTTP MCP server) at runtime.
4. **Validate** — `go vet`/`go test`/plan-compliance. On failure, a scoped
   RPI re-run of just the failing path's owning team (via the lock
   registry), not the whole pipeline.
5. **Code Review** — an automated security/quality pass, pending human
   sign-off.
6. **Commit and Ship** — delegates to a headless worker: `gh pr create`,
   falling back to `git push`/branch handoff.

## Layout

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

## Usage

```
go build ./...
go run ./cmd/ledger run <repo> "<task description>"   # run the pipeline + TUI
go run ./cmd/ledger prune [repo]                       # remove leftover worktrees
```

TUI keybindings: arrows to move the cursor, `space` to select an agent,
`enter` to collapse/expand a phase, `k`ill (confirm), `p`ause/resume,
`b`/btw (kill + relay a message), `/` to search agents by name/id. Review
gates (after Research/Plan/Review) render as a modal: `a`pprove, `r`eject,
`e`dit-comment.

## Status

M0–M10 of IMPLEMENTATION_PLAN.md are implemented, including model routing
(M8) and the Bubble Tea TUI (M9), each with tests exercising the real
`claude` CLI end to end (skipped automatically if `claude` isn't on PATH).

Known gaps, deliberately scoped rather than silently missing:
- `/btw` kills and journals the relayed message but doesn't yet auto-respawn
  the role with the message prepended — that needs a per-role restart hook
  the phases fan-out loops don't expose yet (see the `ponytail:` comment on
  `tui.Model.handleBtwKey`).
- `Pause`/`Resume` are logical/UI-visible only, not a real OS process
  suspend (no portable SIGSTOP equivalent via `os/exec` on Windows).
- `cmd/ledger-ownership-mcp` (the standalone stdio MCP binary from M4) isn't
  used by `Implement` anymore, which shares one in-process registry over
  HTTP instead to avoid multi-process write races; kept as a dev tool.

## Out of scope (v1)

Ring/multi-orchestrator coordination, user-configurable pipeline shapes,
live registry editing via the TUI, detach/reattach (OS job control only).
