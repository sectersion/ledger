import { spawn, type ChildProcess } from "node:child_process";
import { createInterface } from "node:readline";

export type StreamEvent = Record<string, unknown> & { type: string };

/** Running workers by agentId, so failure-retry and /btw can kill by id. */
const activeWorkers = new Map<string, ChildProcess>();

/** Kills the tracked worker for agentId, if still running. Returns whether one was found. */
export function killWorker(agentId: string): boolean {
  const child = activeWorkers.get(agentId);
  if (!child) return false;
  child.kill();
  return true;
}

/** Captures the final "result" event's text as workers run; pass as onEvent to spawnWorker. */
export function collectResult(): { onEvent: (e: StreamEvent) => void; text: () => string } {
  let text = "";
  return {
    onEvent: (e) => {
      if (e.type === "result" && typeof e.result === "string") text = e.result;
    },
    text: () => text,
  };
}

export function spawnWorker(
  cwd: string,
  taskPrompt: string,
  onEvent: (event: StreamEvent) => void,
  extraArgs: string[] = [],
  agentId?: string,
): Promise<number> {
  return new Promise((resolve, reject) => {
    const child = spawn(
      "claude",
      ["-p", taskPrompt, "--output-format", "stream-json", "--verbose", ...extraArgs],
      { cwd, stdio: ["ignore", "pipe", "inherit"] },
    );

    if (agentId) activeWorkers.set(agentId, child);

    const lines = createInterface({ input: child.stdout });
    lines.on("line", (line) => {
      if (!line.trim()) return;
      try {
        onEvent(JSON.parse(line) as StreamEvent);
      } catch {
        // non-JSON line from the CLI (e.g. a warning) — ignore
      }
    });

    const cleanup = () => {
      if (agentId && activeWorkers.get(agentId) === child) activeWorkers.delete(agentId);
    };

    child.on("error", (err) => {
      cleanup();
      reject(err);
    });
    child.on("close", (code) => {
      cleanup();
      resolve(code ?? 0);
    });
  });
}
