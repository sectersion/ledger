import { readdir } from "node:fs/promises";
import { readJournal, type JournalRecord } from "./journal.ts";
import { loadRegistry, type Registry } from "./registry.ts";

export type AgentState = {
  agentId: string;
  phase: string | null;
  status: "started" | "completed" | "failed" | null;
  entries: JournalRecord[];
};

export type ResumeState = {
  agents: AgentState[];
  registry: Registry;
};

export async function resumeState(ledgerDir: string): Promise<ResumeState> {
  const registry = await loadRegistry(ledgerDir);

  let agentIds: string[];
  try {
    agentIds = (await readdir(`${ledgerDir}/journals`))
      .filter((f) => f.endsWith(".jsonl"))
      .map((f) => f.slice(0, -".jsonl".length));
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") agentIds = [];
    else throw err;
  }

  const agents = await Promise.all(
    agentIds.map(async (agentId): Promise<AgentState> => {
      const entries = await readJournal(ledgerDir, agentId);
      const lastPhaseEntry = [...entries].reverse().find((e) => e.type === "phase");
      return {
        agentId,
        phase: lastPhaseEntry?.type === "phase" ? lastPhaseEntry.phase : null,
        status: lastPhaseEntry?.type === "phase" ? lastPhaseEntry.status : null,
        entries,
      };
    }),
  );

  return { agents, registry };
}
