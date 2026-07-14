import { readFile, mkdir, writeFile } from "node:fs/promises";
import { spawnWorker, collectResult } from "../worker.ts";
import { appendJournal } from "../journal.ts";

const PLAN_ROLES = ["Architecture Planner", "Risk Analyzer", "Test Planner"];

/** Fans out planner roles against the research report and writes a spec+rubric plan file. */
export async function runPlan(
  repoPath: string,
  ledgerDir: string,
  task: string,
  researchReportPath: string,
): Promise<string> {
  const agentId = "plan";
  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "plan", status: "started" });

  const researchReport = await readFile(researchReportPath, "utf8");

  const sections = await Promise.all(
    PLAN_ROLES.map(async (role) => {
      const result = collectResult();
      const prompt = `You are the ${role} subagent for a planning phase. Task: ${task}\n\nResearch report:\n${researchReport}\n\nProduce your section of the implementation plan only, within your remit.`;
      await spawnWorker(repoPath, prompt, result.onEvent);
      return `## ${role}\n\n${result.text() || "(no output)"}\n`;
    }),
  );

  const plan = `# Plan\n\nTask: ${task}\n\n${sections.join("\n")}`;
  const artifactsDir = `${ledgerDir}/artifacts`;
  await mkdir(artifactsDir, { recursive: true });
  const planPath = `${artifactsDir}/plan.md`;
  await writeFile(planPath, plan);

  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "plan", status: "completed" });
  return planPath;
}
