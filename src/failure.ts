import { killWorker } from "./worker.ts";
import { runRole, type ImplementRole } from "./phases/implement.ts";
import { appendJournal } from "./journal.ts";
import { runGate } from "./gates.ts";

const MAX_RETRIES = 2;

/** Kills the tracked worker for agentId (if running) and respawns via respawn(). The shared kill-and-respawn primitive for both failure-retry and /btw. */
export function killAndRespawn<T>(agentId: string, respawn: () => Promise<T>): Promise<T> {
  killWorker(agentId);
  return respawn();
}

/**
 * Runs a role, and on failure kills its worker and respawns a fresh team for that section
 * (fresh worktree/journal entries) up to MAX_RETRIES times. Past the cap, escalates as a
 * pending gate; approving the gate grants one more retry, rejecting gives up.
 */
export async function runRoleWithRetries(
  repoPath: string,
  ledgerDir: string,
  planPath: string,
  role: ImplementRole,
): Promise<"pass" | "fail"> {
  let attempts = 0;
  for (;;) {
    const verdict = await killAndRespawn(role.agentId, () =>
      runRole(repoPath, ledgerDir, planPath, role).catch(() => "fail" as const),
    );
    if (verdict === "pass") return "pass";

    attempts++;
    if (attempts <= MAX_RETRIES) continue;

    const gateName = `escalate:${role.agentId}`;
    await appendJournal(ledgerDir, role.agentId, {
      type: "error",
      message: `${role.role} failed ${attempts} times; escalating for review`,
    });
    const decision = await runGate(gateName);
    if (decision.decision !== "approve") return "fail";
    attempts = 0;
  }
}

/** /btw relay: kills the target worker and respawns it with the relayed message prepended to its task. */
export function relayToWorker(
  repoPath: string,
  ledgerDir: string,
  planPath: string,
  role: ImplementRole,
  message: string,
): Promise<"pass" | "fail"> {
  const relayedRole: ImplementRole = { ...role, task: `${message}\n\n${role.task}` };
  return killAndRespawn(role.agentId, () => runRole(repoPath, ledgerDir, planPath, relayedRole));
}
