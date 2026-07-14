import { runResearch } from "./phases/research.ts";
import { runPlan } from "./phases/plan.ts";
import { runImplement, type ImplementRole } from "./phases/implement.ts";
import { runValidate } from "./phases/validate.ts";
import { runReview } from "./phases/review.ts";
import { runShip } from "./phases/ship.ts";
import { runGate } from "./gates.ts";

const MAX_VALIDATE_RETRIES = 2;

export async function runPipeline(
  repoPath: string,
  ledgerDir: string,
  task: string,
  roles: ImplementRole[],
): Promise<void> {
  const researchReportPath = await runResearch(repoPath, ledgerDir, task);
  await runGate("research");

  const planPath = await runPlan(repoPath, ledgerDir, task, researchReportPath);
  await runGate("plan");

  await runImplement(repoPath, ledgerDir, planPath, roles);

  for (const role of roles) {
    let result = await runValidate(role.worktreePath, ledgerDir, role.agentId, planPath);
    let retries = 0;
    while (!result.pass && retries < MAX_VALIDATE_RETRIES) {
      await runImplement(repoPath, ledgerDir, planPath, [role]);
      result = await runValidate(role.worktreePath, ledgerDir, role.agentId, planPath);
      retries++;
    }
  }

  for (const role of roles) {
    await runReview(role.worktreePath, ledgerDir, role.agentId, planPath);
  }

  for (const role of roles) {
    await runShip(role.worktreePath, ledgerDir, role.agentId);
  }
}
