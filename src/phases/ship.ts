import { spawnWorker, collectResult } from "../worker.ts";
import { appendJournal } from "../journal.ts";

const SHIP_PROMPT = `Ship the current changes on this branch: try "gh pr create" (fill in a sensible title/body from the diff/commits). If gh is not installed or the command fails, fall back to "git push -u origin HEAD" and report the branch name so a human can open a PR manually. Report exactly what you ran and the outcome.`;

/** Delegates commit/ship mechanics to a worker in the given worktree. */
export async function runShip(
  worktreePath: string,
  ledgerDir: string,
  agentId: string,
): Promise<{ code: number; summary: string }> {
  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "ship", status: "started" });

  const result = collectResult();
  const code = await spawnWorker(worktreePath, SHIP_PROMPT, result.onEvent);
  const summary = result.text();

  await appendJournal(ledgerDir, agentId, { type: "diff", summary: summary || `exit code ${code}` });
  await appendJournal(ledgerDir, agentId, {
    type: "phase",
    phase: "ship",
    status: code === 0 ? "completed" : "failed",
  });

  return { code, summary };
}
