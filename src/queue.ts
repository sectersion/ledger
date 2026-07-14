import { loadSettings, DEFAULT_CONCURRENCY_CAP } from "./settings.ts";

export type Phase = "research" | "plan" | "implement" | "validate" | "review" | "ship";
export type TaskStatus = "queued" | "running" | "completed" | "failed";

export interface Task {
  id: string;
  phase: Phase;
  status: TaskStatus;
}

const DEFAULT_CAP = DEFAULT_CONCURRENCY_CAP;

async function readConcurrencyCap(): Promise<number> {
  const settings = await loadSettings();
  return settings.concurrencyCap ?? DEFAULT_CAP;
}

export class TaskQueue {
  private pending: Task[] = [];
  private running = 0;
  private cap = DEFAULT_CAP;
  private capReady: Promise<void>;

  constructor(cap?: number) {
    this.capReady = cap !== undefined
      ? Promise.resolve().then(() => { this.cap = cap; })
      : readConcurrencyCap().then((c) => { this.cap = c; });
  }

  enqueue(task: Task): void {
    this.pending.push(task);
  }

  get runningCount(): number {
    return this.running;
  }

  get pendingCount(): number {
    return this.pending.length;
  }

  /** Runs all enqueued tasks against executor, respecting the concurrency cap, until the queue drains. */
  async drain(executor: (task: Task) => Promise<void>): Promise<void> {
    await this.capReady;
    return new Promise((resolve, reject) => {
      let failed: unknown;
      const pump = () => {
        if (failed) return;
        while (this.running < this.cap && this.pending.length > 0) {
          const task = this.pending.shift()!;
          this.running++;
          task.status = "running";
          executor(task)
            .then(() => { task.status = "completed"; })
            .catch((err) => {
              task.status = "failed";
              failed = err;
            })
            .finally(() => {
              this.running--;
              if (failed) {
                reject(failed);
                return;
              }
              if (this.running === 0 && this.pending.length === 0) {
                resolve();
              } else {
                pump();
              }
            });
        }
        if (this.running === 0 && this.pending.length === 0) resolve();
      };
      pump();
    });
  }
}
