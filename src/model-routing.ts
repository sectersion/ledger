import { spawnWorker, collectResult } from "./worker.ts";
import { loadSettings, DEFAULT_MODEL_ALLOW_LIST, type Settings } from "./settings.ts";

export type { Settings };
export const readSettings = loadSettings;

const DEFAULT_ALLOW_LIST = DEFAULT_MODEL_ALLOW_LIST;

/** One-off routing call: asks `claude -p` which model best fits taskDescription, then clamps the answer to settings' allow-list. */
export async function chooseModel(taskDescription: string, settings: Settings): Promise<string> {
  const allowList = settings.modelAllowList ?? DEFAULT_ALLOW_LIST;

  const result = collectResult();
  await spawnWorker(
    process.cwd(),
    `You are choosing which Claude model to use for this task, from this exact list: ${allowList.join(", ")}. ` +
      `Task: ${taskDescription}\n\nReply with ONLY the model id from the list, nothing else.`,
    result.onEvent,
  );

  const suggestion = result.text().trim();
  return allowList.includes(suggestion) ? suggestion : allowList[0];
}
