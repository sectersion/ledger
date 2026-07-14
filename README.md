# ledger

An orchestration layer outside Claude Code. A single orchestrator process
drives headless Claude Code instances as workers to execute large tasks
(security audits, framework migrations, full-codebase overhauls) through a
fixed 6-phase pipeline, with git worktree isolation, per-agent journals, and
human review gates.

See [PLAN.md](PLAN.md) for the full design and [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)
for how it was built, milestone by milestone.

## Stack

Node + TypeScript (ESM), Ink for the TUI, the `claude` CLI spawned as a
subprocess (no SDK), plain JSON/JSONL files for state (no DB).

## Pipeline

1. **Research** — fixed subagent roles fan out, aggregate into a compressed
   report. Blocking human review gate.
2. **Plan** — Architecture/Risk/Test planner roles produce a spec + rubric.
   Blocking human review gate.
3. **Implement** — Backend/Frontend/Test workers execute the plan, each in
   its own git worktree, requesting file ownership from the orchestrator at
   runtime.
4. **Validate** — tests/linters/plan-compliance. On failure, a scoped
   re-run of just the failing worktree/team (not the whole pipeline).
5. **Code Review** — automated checks + human sign-off gate.
6. **Commit and Ship** — delegates to a CC instance: `gh pr create`, falling
   back to `git push`/branch handoff.

## Layout

```
src/
  worker.ts              spawn a headless claude CLI worker, stream stream-json events
  worktree.ts            git worktree add/remove wrappers
  spawn-in-worktree.ts    wires worker spawn to a fresh worktree
  journal.ts             per-agent append-only JSONL journal
  registry.ts            path -> owning agentId lock registry
  resume.ts              reconstruct orchestrator state from journals + registry
  queue.ts               in-memory FIFO task queue with a concurrency cap
  gates.ts               pending human-review gate map (research/plan/review)
  failure.ts             kill-and-respawn, retry cap + escalation, /btw relay
  model-routing.ts       one-off "best model for this job" routing
  settings.ts            ~/.ledger/settings.json loader
  mcp/                   ownership MCP server + --mcp-config generator
  phases/                research, plan, implement, validate, review, ship
  pipeline.ts            wires the 6 phases together
  tui/                   Ink split-pane UI (agent tree + journal stream)
  cli/prune.ts           explicit worktree prune command
```

## Usage

```
npm install
npm run dev              # launch the TUI (demo state if no LEDGER_DIR set)
LEDGER_DIR=<dir> npm run dev   # launch against real orchestrator state
npm run build             # tsc -> dist/
npm run prune <repoPath> <worktreePath>   # remove a worktree explicitly
```

## Status

M0–M10 of IMPLEMENTATION_PLAN.md are implemented. Known gaps: no ESLint
config yet (`npm run lint` needs `eslint.config.js`), no persistent
regression suite beyond `src/tui/App.test.tsx` and
`src/e2e-dry-run.manual-test.ts`, and model-routing/`/btw` aren't yet wired
into a single live orchestrator loop end to end.

## Out of scope (v1)

Ring/multi-orchestrator coordination, user-configurable pipeline shapes,
live registry editing via the TUI, detach/reattach (OS job control only).
