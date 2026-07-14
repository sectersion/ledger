import { mkdir, writeFile } from "node:fs/promises";
import { spawnWorker, collectResult } from "../worker.ts";
import { appendJournal } from "../journal.ts";

const RESEARCH_ROLES = [
  "Codebase Locator",
  "Codebase Analyzer",
  "Pattern Finder",
  "External Research",
  "Historical Context",
];

/** Fans out fixed research roles against repoPath (no worktree needed — research is read-only) and writes a compressed report. */
export async function runResearch(
  repoPath: string,
  ledgerDir: string,
  task: string,
): Promise<string> {
  const agentId = "research";
  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "research", status: "started" });

  const sections = await Promise.all(
    RESEARCH_ROLES.map(async (role) => {
      const result = collectResult();
      const prompt = `You are the ${role} subagent for a research phase. Task: ${task}\nInvestigate the current repository and report concise, factual findings relevant to your role only. No fluff, no recommendations outside your remit.`;
      await spawnWorker(repoPath, prompt, result.onEvent);
      return `## ${role}\n\n${result.text() || "(no output)"}\n`;
    }),
  );

  const report = `# Research Report\n\nTask: ${task}\n\n${sections.join("\n")}`;
  const artifactsDir = `${ledgerDir}/artifacts`;
  await mkdir(artifactsDir, { recursive: true });
  const reportPath = `${artifactsDir}/research-report.md`;
  await writeFile(reportPath, report);

  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "research", status: "completed" });
  return reportPath;
}
