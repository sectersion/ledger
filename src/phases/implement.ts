import { mkdir, writeFile } from "node:fs/promises";
import { createWorktree } from "../worktree.ts";
import { spawnWorker } from "../worker.ts";
import { appendJournal } from "../journal.ts";
import { requestOwnership, releaseOwnership } from "../registry.ts";
import { ownershipMcpConfig } from "../mcp/ownership-config.ts";

export type ImplementRole = {
  role: string;
  agentId: string;
  branch: string;
  worktreePath: string;
  paths: string[];
  task: string;
};

/** Spawns one role's worker in its own worktree, pre-granting ownership of its paths via the registry. */
export async function runRole(
  repoPath: string,
  ledgerDir: string,
  planPath: string,
  r: ImplementRole,
): Promise<"pass" | "fail"> {
  await appendJournal(ledgerDir, r.agentId, { type: "phase", phase: "implement", status: "started" });

  const grants: string[] = [];
  for (const p of r.paths) {
    const granted = await requestOwnership(ledgerDir, p, r.agentId);
    if (granted) {
      grants.push(p);
      await appendJournal(ledgerDir, r.agentId, { type: "ownership", action: "granted", path: p });
    } else {
      await appendJournal(ledgerDir, r.agentId, { type: "error", message: `ownership denied for ${p}` });
    }
  }

  // ponytail: re-running a role (validate-fail retry) hits an existing worktree; reuse it rather
  // than reimplementing worktree-exists detection. Full retry semantics land in M7.
  const cwd = await createWorktree(repoPath, r.branch, r.worktreePath).catch(() => r.worktreePath);

  const mcpConfigPath = `${ledgerDir}/mcp/${r.agentId}.json`;
  await mkdir(`${ledgerDir}/mcp`, { recursive: true });
  await writeFile(mcpConfigPath, JSON.stringify(ownershipMcpConfig(ledgerDir, r.agentId)));

  const prompt = `You are the ${r.role} implementer. Task: ${r.task}\nFollow the plan at ${planPath}. You own these paths: ${grants.join(", ") || "(none)"}. Make the necessary changes in this worktree.`;

  let verdict: "pass" | "fail" = "fail";
  try {
    const code = await spawnWorker(cwd, prompt, () => {}, ["--mcp-config", mcpConfigPath], r.agentId);
    verdict = code === 0 ? "pass" : "fail";
    await appendJournal(ledgerDir, r.agentId, {
      type: "verdict",
      verdict,
      detail: `exit code ${code}`,
    });
  } finally {
    for (const p of grants) {
      await releaseOwnership(ledgerDir, p, r.agentId);
      await appendJournal(ledgerDir, r.agentId, { type: "ownership", action: "released", path: p });
    }
    await appendJournal(ledgerDir, r.agentId, { type: "phase", phase: "implement", status: "completed" });
  }
  return verdict;
}

/** Spawns one worker per role, each in its own worktree, pre-granting ownership of its paths via the registry. */
export async function runImplement(
  repoPath: string,
  ledgerDir: string,
  planPath: string,
  roles: ImplementRole[],
): Promise<void> {
  for (const r of roles) {
    await runRole(repoPath, ledgerDir, planPath, r);
  }
}
