import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { readFile } from "node:fs/promises";
import { spawnWorker, collectResult } from "../worker.ts";
import { appendJournal } from "../journal.ts";

const execFileAsync = promisify(execFile);

async function hasScript(worktreePath: string, script: string): Promise<boolean> {
  try {
    const pkg = JSON.parse(await readFile(`${worktreePath}/package.json`, "utf8")) as {
      scripts?: Record<string, string>;
    };
    return Boolean(pkg.scripts?.[script]);
  } catch {
    return false;
  }
}

async function runScript(worktreePath: string, script: string): Promise<{ ok: boolean; detail: string }> {
  try {
    const { stdout } = await execFileAsync("npm", ["run", script], { cwd: worktreePath });
    return { ok: true, detail: stdout.slice(-2000) };
  } catch (err) {
    const e = err as { stdout?: string; stderr?: string; message: string };
    return { ok: false, detail: (e.stderr ?? e.stdout ?? e.message).slice(-2000) };
  }
}

export type ValidateResult = { pass: boolean; failingAgentId?: string; detail: string };

/** Runs tests/lint (if present) plus a plan-compliance check for a single worktree/role. */
export async function runValidate(
  worktreePath: string,
  ledgerDir: string,
  agentId: string,
  planPath: string,
): Promise<ValidateResult> {
  await appendJournal(ledgerDir, agentId, { type: "phase", phase: "validate", status: "started" });

  const details: string[] = [];
  let pass = true;

  for (const script of ["test", "lint"] as const) {
    if (!(await hasScript(worktreePath, script))) continue;
    const result = await runScript(worktreePath, script);
    details.push(`npm run ${script}: ${result.ok ? "pass" : "fail"}`);
    if (!result.ok) pass = false;
  }

  if (pass) {
    const plan = await readFile(planPath, "utf8").catch(() => "(no plan found)");
    let diff = "";
    try {
      diff = (await execFileAsync("git", ["diff", "HEAD"], { cwd: worktreePath })).stdout;
    } catch {
      diff = "(no diff)";
    }
    const result = collectResult();
    await spawnWorker(
      worktreePath,
      `Compare this diff against the plan below. Reply with exactly PASS or FAIL on the first line, followed by a one-line reason.\n\nPlan:\n${plan}\n\nDiff:\n${diff}`,
      result.onEvent,
    );
    const compliance = result.text();
    details.push(`plan-compliance: ${compliance.slice(0, 200)}`);
    if (!compliance.trim().toUpperCase().startsWith("PASS")) pass = false;
  }

  const detail = details.join("; ");
  await appendJournal(ledgerDir, agentId, { type: "verdict", verdict: pass ? "pass" : "fail", detail });
  await appendJournal(ledgerDir, agentId, {
    type: "phase",
    phase: "validate",
    status: pass ? "completed" : "failed",
  });

  return { pass, failingAgentId: pass ? undefined : agentId, detail };
}
