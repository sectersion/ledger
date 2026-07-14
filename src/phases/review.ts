import { appendJournal } from "../journal.ts";
import { runValidate, type ValidateResult } from "./validate.ts";
import { runGate, type GateDecision } from "../gates.ts";

export type ReviewResult = { automated: ValidateResult; decision: GateDecision };

/** Runs automated checks, then blocks on a human sign-off gate resolved externally via resolveGate(gateName, ...). */
export async function runReview(
  worktreePath: string,
  ledgerDir: string,
  agentId: string,
  planPath: string,
): Promise<ReviewResult> {
  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "review", status: "started" });

  const automated = await runValidate(worktreePath, ledgerDir, agentId, planPath);

  const gateName = `review:${agentId}`;
  const decision = await runGate(gateName);

  await appendJournal(ledgerDir, agentId, {
    type: "phase",
    phase: "review",
    status: decision.decision === "approve" ? "completed" : "failed",
  });

  return { automated, decision };
}
