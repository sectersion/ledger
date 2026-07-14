import { mkdir, writeFile, rm } from "node:fs/promises";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import assert from "node:assert/strict";
import os from "node:os";
import { runResearch } from "./phases/research.ts";
import { runPlan } from "./phases/plan.ts";
import { runRole, type ImplementRole } from "./phases/implement.ts";
import { runValidate } from "./phases/validate.ts";
import { resumeState } from "./resume.ts";
import { loadRegistry } from "./registry.ts";

const execFileAsync = promisify(execFile);
const scratch = `${os.tmpdir()}/ledger-e2e-${Date.now()}`;
const repoPath = `${scratch}/repo`;
const worktreePath = `${scratch}/worktree-a`;
const ledgerDir = `${scratch}/ledger`;

async function main() {
  await mkdir(repoPath, { recursive: true });
  await execFileAsync("git", ["init"], { cwd: repoPath });
  await execFileAsync("git", ["config", "user.email", "test@test.com"], { cwd: repoPath });
  await execFileAsync("git", ["config", "user.name", "test"], { cwd: repoPath });
  await writeFile(
    `${repoPath}/package.json`,
    JSON.stringify({ name: "toy", version: "1.0.0", scripts: { test: "exit 1" } }, null, 2),
  );
  await writeFile(`${repoPath}/README.md`, "toy repo\n");
  await execFileAsync("git", ["add", "-A"], { cwd: repoPath });
  await execFileAsync("git", ["commit", "-m", "init"], { cwd: repoPath });

  const task = "Add a one-line comment to README.md";

  const researchReportPath = await runResearch(repoPath, ledgerDir, task);
  const planPath = await runPlan(repoPath, ledgerDir, task, researchReportPath);
  assert.ok(researchReportPath && planPath, "research/plan artifacts written");
  console.log("research + plan phases produced artifacts");

  const role: ImplementRole = {
    role: "Backend",
    agentId: "backend",
    branch: "e2e-backend",
    worktreePath,
    paths: ["README.md"],
    task,
  };

  await runRole(repoPath, ledgerDir, planPath, role);

  let result = await runValidate(worktreePath, ledgerDir, role.agentId, planPath);
  let retries = 0;
  const MAX_VALIDATE_RETRIES = 2;
  while (!result.pass && retries < MAX_VALIDATE_RETRIES) {
    await runRole(repoPath, ledgerDir, planPath, role);
    result = await runValidate(worktreePath, ledgerDir, role.agentId, planPath);
    retries++;
  }

  assert.equal(result.pass, false, "npm test always fails: validate should never pass");
  assert.equal(retries, MAX_VALIDATE_RETRIES, "scoped re-run loop should exhaust both retries");
  console.log(`validate failed as induced, scoped re-run fired ${retries} times`);

  const midState = await resumeState(ledgerDir);
  const backendAgent = midState.agents.find((a) => a.agentId === "backend");
  assert.ok(backendAgent, "resumeState sees the backend agent from on-disk journal");
  const validateEntries = backendAgent!.entries.filter(
    (e) => e.type === "phase" && e.phase === "validate",
  );
  assert.equal(validateEntries.length, (MAX_VALIDATE_RETRIES + 1) * 2, "started+completed per validate attempt");
  const failedVerdicts = backendAgent!.entries.filter((e) => e.type === "verdict" && e.verdict === "fail");
  assert.equal(failedVerdicts.length, MAX_VALIDATE_RETRIES + 1, "every validate attempt recorded a fail verdict");
  console.log("resumeState mid-run matches on-disk journal");

  const registryOnDisk = await loadRegistry(ledgerDir);
  assert.deepEqual(midState.registry, registryOnDisk, "resumeState registry matches loadRegistry directly");
  assert.deepEqual(registryOnDisk, {}, "ownership released after each implement attempt, registry back to empty");
  console.log("resumeState registry matches fresh loadRegistry() read (crash-resume proof)");

  console.log("ALL PASS");
}

main()
  .catch((err) => {
    console.error(err);
    process.exitCode = 1;
  })
  .finally(async () => {
    await rm(scratch, { recursive: true, force: true }).catch(() => {});
  });
