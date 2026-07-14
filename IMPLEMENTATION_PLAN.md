# ledger — Implementation Plan

Derived from PLAN.md. Ordered so each milestone is runnable/testable before
the next starts — no big-bang integration at the end.

Stack: Node + TypeScript, Ink (TUI), `claude` CLI spawned as subprocess
(no SDK), plain JSON files for registry/journals (no DB — matches PLAN.md's
"plain process" stance).

## M0 — Project skeleton
- `package.json`, TS config, entry point (`src/index.tsx`), lint/test runner.
- No logic yet; just `ledger` boots an empty Ink screen.

## M1 — Worker spawn primitive
- `spawnWorker(cwd, taskPrompt)`: runs `claude -p <prompt> --output-format
  stream-json` as a subprocess in a given cwd, streams stdout.
- Parse `stream-json` lines into typed events as they arrive (no polling).
- Manual test: spawn one worker against a scratch dir, confirm events stream.
- No worktrees, no orchestrator yet — just prove the subprocess/stream contract.

## M2 — Git worktree lifecycle
- `createWorktree(branch)` / `pruneWorktree(path)` wrapping `git worktree add`
  / `git worktree remove`.
- Wire into M1: orchestrator creates the worktree, passes path as spawn cwd.
- Worktrees left on disk after use; prune is a separate explicit command.

## M3 — Journals + lock registry (the two on-disk sources of truth)
- Per-agent journal file: append-only JSON lines — diff produced, verdict,
  ownership grants/releases, errors, and phase/status entries (single
  source of truth for phase state, per PLAN.md's resume section).
- Lock registry JSON: path → current owner, orchestrator-owned, no shared
  writes.
- These two files + git checkpoints are all that's needed to reconstruct
  state on orchestrator restart — build resume logic against this pair now
  before adding phases, so it's tested early rather than bolted on.

## M4 — Ownership MCP server
- Small MCP server exposing `request_ownership(path)` / `release_ownership(path)`,
  checked against the lock registry (M3).
- Passed to each worker via `--mcp-config`.
- Manual test: two workers requesting overlapping paths — one gets denied.

## M5 — DAG + concurrency queue
- In-memory FIFO task queue, concurrency cap (default 10, from
  `~/.ledger/settings.json`).
- DAG state tracks phase membership per task; queued-but-unspawned tasks are
  the only state that's *not* durable (by design — nothing lost on crash
  since they have no worktree/journal yet).

## M6 — Pipeline phases (built one at a time, in order, each testable
  against a toy repo before the next):
1. Research — fan out fixed research subagent roles, aggregate into one
   report file.
2. Plan — fan out planner roles, produce spec+rubric file.
3. Implement — spawn Backend/Frontend/Test workers, exercise ownership
   requests (M4) for real.
4. Validate — run tests/linters/plan-compliance; on failure, scope a
   re-run to just the worktree/team that owned the failing path (per lock
   registry, no dependency graph needed).
5. Code Review — automated checks + human sign-off gate.
6. Commit and Ship — delegate to a CC instance: try `gh pr create`, fall
   back to `git push`/branch handoff.
- Research/Plan/Ship gates block the orchestrator entirely (sequential
  phases, nothing else to keep moving).

## M7 — Failure handling
- Worker failure: kill worker + context, respawn fresh team, cap 2 retries,
  then escalate (surface as a gate-like blocking notification).
- `/btw`: kill-and-respawn with the relayed message prepended to task
  context (reuses M7's respawn primitive — no new mechanism).

## M8 — Model routing
- Orchestrator's own one-off `claude` call ("best model for this job"),
  filtered by settings.json allow-list. Wire in after phases exist, since
  it's a routing decision per spawn, not new spawn machinery.

## M9 — TUI (Ink)
- Split layout: collapsible agent tree (left) + live journal stream
  (right), top/bottom bars per PLAN.md's mockup.
- Cursor nav + single-select; k/p/`/btw` act on selected agent.
- Kill = confirm modal; pause = immediate.
- Review gates = modal overlay (not pane swap), a/r/e keybindings live at
  all times.
- Toast/status-line for phase transitions and non-blocking auto-requeue.
- `/` search filters tree by agent name/id only.
- Build this last: it's a renderer over state M1–M8 already produce: no
  TUI-only state to invent.

## M10 — Settings & polish
- `~/.ledger/settings.json`: concurrency cap, model allow-list, max
  thinking levels.
- Explicit prune command for worktrees.
- End-to-end dry run: toy repo through all 6 phases with induced failures
  at Validate to confirm scoped RPI re-run and resume-from-crash both work.

## Out of scope (per PLAN.md, don't build)
- Ring/multi-orchestrator coordination.
- User-configurable pipeline shapes.
- Live registry editing via TUI.
- Detach/reattach — single process, OS job control only.

## Suggested build order rationale
M0-M3 establish the two durable data sources (journals, registry) before
anything depends on them. M4-M5 add the runtime coordination primitives.
M6 is the bulk of the product — built phase-by-phase against a toy repo so
each is independently testable. M7-M8 are cross-cutting concerns layered on
top of working phases. M9 (TUI) is deliberately last: it only renders state
the backend already produces, so building it early would mean building it
twice.
