import type { AgentState, ResumeState } from "../resume.ts";

/** Standalone demo dataset for running the TUI with no ledgerDir given. */
export function demoState(): ResumeState {
  const agents: AgentState[] = [
    {
      agentId: "codebase-locator",
      phase: "research",
      status: "completed",
      entries: [
        { type: "phase", phase: "research", status: "started", timestamp: "2026-07-14T12:00:00Z" },
        { type: "diff", summary: "wrote research/locator.md", timestamp: "2026-07-14T12:01:00Z" },
        { type: "phase", phase: "research", status: "completed", timestamp: "2026-07-14T12:02:00Z" },
      ],
    },
    {
      agentId: "backend-3",
      phase: "implement",
      status: "started",
      entries: [
        { type: "phase", phase: "implement", status: "started", timestamp: "2026-07-14T12:04:00Z" },
        { type: "ownership", action: "granted", path: "src/api/routes.ts", timestamp: "2026-07-14T12:04:10Z" },
        { type: "diff", summary: "writing src/api/routes.ts", timestamp: "2026-07-14T12:04:33Z" },
      ],
    },
    {
      agentId: "frontend-2",
      phase: "implement",
      status: null,
      entries: [],
    },
  ];
  return { agents, registry: { "src/api/routes.ts": "backend-3" } };
}

export const DEMO_GATE_ARTIFACT = `# Research Report (demo)

- Codebase uses Ink + React for the TUI.
- Journals are append-only JSONL, one file per agent.
- Lock registry is a single JSON file, orchestrator-owned.
`;
