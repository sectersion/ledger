import { createWorktree } from "./worktree.ts";
import { spawnWorker, type StreamEvent } from "./worker.ts";

export async function spawnWorkerInWorktree(
  repoPath: string,
  branch: string,
  worktreePath: string,
  taskPrompt: string,
  onEvent: (event: StreamEvent) => void,
): Promise<number> {
  const cwd = await createWorktree(repoPath, branch, worktreePath);
  return spawnWorker(cwd, taskPrompt, onEvent);
}
