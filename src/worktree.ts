import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

export async function createWorktree(
  repoPath: string,
  branch: string,
  worktreePath: string,
): Promise<string> {
  await execFileAsync(
    "git",
    ["worktree", "add", "-b", branch, worktreePath],
    { cwd: repoPath },
  );
  return worktreePath;
}

export async function pruneWorktree(
  repoPath: string,
  worktreePath: string,
): Promise<void> {
  await execFileAsync(
    "git",
    ["worktree", "remove", worktreePath, "--force"],
    { cwd: repoPath },
  );
}
