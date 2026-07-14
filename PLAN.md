# ledger — v1 Plan

An orchestration layer outside Claude Code. Instead of working directly in a
Claude Code terminal session, a single orchestrator process drives headless
Claude Code instances as workers to execute large tasks (security audits,
framework migrations, full-codebase overhauls) through a fixed pipeline, with
git worktree isolation, per-agent journals, and human review gates.

## Architecture

- Single orchestrator (no ring/democratic model — deferred past v1).
- Orchestrator is a plain process (Go, TUI built with Bubble Tea), not itself
  a Claude agent — it's deterministic control flow: DAG state, spawning
  worktrees, granting locks, aggregating journals.
- Orchestrator spawns all work as headless Claude Code instances, each in
  its own git worktree.

## Pipeline (fixed, hardcoded for v1)

1. **Research** (compression of truth) — coordinator fans out parallel
   modules: Codebase Locator, Codebase Analyzer, Pattern Finder, External
   Research, Historical Context. Produces a compressed research report.
   → **Human review gate (blocking).**
2. **Plan** (compression of intent) — Architecture Planner, Risk Analyzer,
   Test Planner produce an implementation plan (spec + rubric), saved to
   disk.
   → **Human review gate (blocking).**
3. **Implement** (mechanical execution) — Backend, Frontend, Test workers
   execute the plan. File ownership is not pre-assigned in the plan; each
   worker/team requests ownership from the orchestrator at runtime.
4. **Validate** — verify output against the plan (not just tests): tests,
   linters, plan-compliance checks. On failure, feed back into a scoped
   **RPI (Research → Plan → Implement) re-run of just the failing slice**,
   not the whole pipeline.
5. **Code Review** — quality/security focus: automated security audits,
   type safety, bug review, further automated testing, then human
   engineer oversight.
6. **Commit and Ship** — git operations, CI/CD, PR, merge. Requires human
   sign-off.

Pipeline shape is hardcoded for v1 — not user-configurable.

## File ownership

- No upfront static partitioning by the planner.
- Each subagent team (e.g. Backend, Frontend) requests ownership of a
  path/section from the orchestrator; orchestrator grants/denies against
  a live registry.
- Registry is a JSON file, owned solely by the orchestrator (no shared
  writes across workers).

## Journals

- One journal per agent (not one shared interleaved log).
- Middle-detail content: diff produced, verdict (pass/fail), file
  ownership grants/releases, and errors — not a full raw tool-call trace
  (headless CC already has that internally if ever needed).
- Git checkpoints at defined points (e.g. per ownership release / phase
  completion) as independent rollback points, separate from the final
  merge.
- Rendered in the TUI as a collapsible tree, one node per agent.

## Inter-agent communication

- Orchestrator-mediated only — no peer-to-peer worker channels.
- Worker → orchestrator → other worker, relayed via `/btw`.

## Merging

- Worktree changes merge to main **only after Validate passes** for that
  slice — never earlier (avoids landing broken intermediate state, and
  worktrees are cheap to discard vs. unwinding a bad merge).

## Failure handling

- On worker failure: kill the worker and its context entirely (no
  resume/patch-in-place) and respawn a fresh team for that section.
- Up to 2 retries, then escalate to human.

## Resumability

- Orchestrator crash/restart is recoverable: reconstruct state from git
  checkpoints + JSON lock registry + per-agent journals.

## Human control (TUI)

- Blocking review gates after Research, Plan, and sign-off in
  Review/Ship — orchestrator halts entirely until the human responds
  (phases are sequential, so there's no independent work to keep moving
  in parallel).
- Beyond gates: human can kill, pause, or `/btw` any worker at any time.

## Concurrency & model routing

- Concurrency cap: default 10 concurrent workers, configurable in
  `~/.ledger/settings.json`.
- Orchestrator dynamically picks the best model per job at runtime.
  Settings only restrict — allowed models and max thinking levels — they
  don't fix a per-role model assignment.

## Ship phase mechanics

- Ledger does not push/PR autonomously beyond delegating the job: it
  sends git/PR work to a CC instance that tries `gh pr create` first,
  falling back to plain `git push`/branch handoff if `gh` fails or isn't
  available.

## Worktrees

- Left on disk after merge (for inspection/rollback), not deleted
  automatically on success.
- Explicit/manual prune command provided for cleanup.

## Integration plan

- **Invocation**: orchestrator spawns each worker as a subprocess of the
  `claude` CLI in headless mode (no SDK). One process per worktree.
- **Results**: worker's stdout is parsed as `stream-json` and consumed live
  by the orchestrator; the orchestrator persists what it reads into that
  worker's journal file. No file-based polling.
- **Worktree handoff**: orchestrator runs `git worktree add` itself
  (before spawning), then passes the resulting path to the `claude`
  process as its cwd, plus the task/context via CLI arg or piped stdin.
  Worktree creation never happens worker-side.
- **Ownership requests**: orchestrator runs a small MCP server (exposing
  `request_ownership(path)` / `release_ownership(path)`) and passes it to
  the worker via `--mcp-config`. This is the one live, synchronous channel
  into a running worker.
- **`/btw` relay**: headless runs are one-shot and non-interactive — there
  is no live channel to inject a message into a running worker. `/btw`
  reuses the existing kill-and-respawn primitive: kill the target worker,
  prepend the relayed message to its task context, respawn.
- **TUI**: single process — the Bubble Tea TUI is the orchestrator's own
  renderer, not a separate attachable client. No detach/reattach, no
  cross-process state sync.
- **Model routing**: orchestrator has its own one-off `claude` access,
  separate from worker spawns, used to ask "best model for this task"
  per job (filtered by the settings.json allow-list), rather than a
  static per-phase table.
- **Validate failure scope**: the re-run slice is exactly the worktree/
  team that owned the failing file(s) per the lock registry — not a
  downstream dependency subgraph (none is tracked).
- **Source of truth on resume**: per-agent journals carry phase/status
  (as another journal entry type alongside diffs/verdicts/ownership
  events); the JSON lock registry stays ownership-only (path → current
  owner). Nothing duplicates phase state across both files.
- **Concurrency queue**: in-memory FIFO only, not persisted. A task that's
  merely queued (not yet spawned) has no worktree/journal, so a crash
  loses nothing that isn't already re-derivable from the DAG plus
  completed-task journals on restart.

## TUI UI/UX

- **Layout at rest**: split screen — journal tree (collapsible, one node
  per agent, grouped by phase) as a narrow left pane; selected agent's
  live journal stream as the right pane. Top bar mirrors the bottom bar:
  `ledger · stage: <phase> · substage: <detail>`. Bottom bar shows
  concurrency (`workers: n/10`), gate state, and active keybindings.

  ```
  ┌────────────────────────────────────────────────────────────────────┐
  │ ledger  ·  stage: Implement  ·  substage: backend-3 writing…       │
  ├─ AGENTS ───────────────────┬─ agent: backend-3 ────────────────────┤
  │ ▾ Research                 │ worktree: wt/backend-3               │
  │   ✓ codebase-locator       │ status: implementing (running)       │
  │ ▾ Implement                │ model: sonnet-5                      │
  │   ▾ backend-3   ● running  │ [12:04:33] writing src/api/routes.ts │
  │   ▸ frontend-2  ⏳ queued   │ ...                                   │
  ├─────────────────────────────┴───────────────────────────────────────┤
  │ workers: 3/10 · gate: none pending · [k]ill [p]ause [b]tw [/]search │
  └────────────────────────────────────────────────────────────────────┘
  ```

- **Navigation/selection**: arrow keys move a cursor through the tree;
  `space` marks the agent under the cursor as selected (single-target —
  marking a new agent replaces the old mark, no multi-select). `k`ill,
  `p`ause, and `/btw` always act on the currently selected agent — no
  separate target picker.
- **Review gates** (after Research, Plan, and Ship sign-off): render as a
  **popup modal overlay** on top of the same split layout (tree stays
  visible/dimmed behind it) — not a pane swap, not a full-screen
  takeover. Modal shows the artifact (research report / plan spec),
  scrollable with `j`/`k`/arrows. Bottom bar keybindings swap to
  `[a]pprove [r]eject [e]dit-comment`. Decision keys are live at all
  times — no forced scroll-to-end before approve/reject unlocks.
- **`/btw`**: pressing `b` opens a popup modal with a text input to
  compose the relay message. On submit, the target worker is killed and
  respawned with the message prepended to its task context (headless
  runs are one-shot/non-interactive — there is no live channel to inject
  a message into a running worker; `/btw` reuses the same
  kill-and-respawn primitive as failure recovery).
- **Kill**: requires a confirm modal (`kill backend-3? y/n`) before it
  fires — destructive and irreversible (no resume/patch-in-place), bound
  to a single unmodified keypress next to constantly-used navigation
  keys. **Pause** fires immediately, no confirm — reversible.
- **Phase transitions & non-blocking errors** (e.g. a worker fails
  Validate and gets auto-requeued into scoped RPI, or a phase completes
  and the next starts): surface as a transient toast/status-line
  notification that appears then fades, in addition to the permanent
  journal entry — a passive nudge, not a blocking gate.
- **Search** (`/`): filters/jumps the tree cursor by agent name/id only.
  Not full-text journal search — journals are files on disk, greppable
  directly if deeper content search is ever needed.
- **Process model**: single process — the Bubble Tea TUI is the
  orchestrator's own renderer. No explicit detach/reattach support in v1; backgrounding
  is left to standard OS/terminal job control (tmux/screen/`Ctrl+Z`),
  since there's no separate state-sync layer to support it anyway.

## Explicitly deferred past v1

- Ring / multi-orchestrator / democratic coordination.
- User-configurable pipeline shapes (only the one hardcoded 6-phase
  pipeline exists).
- Live registry editing via TUI (only kill/pause/`/btw`, no manual
  registry surgery).
