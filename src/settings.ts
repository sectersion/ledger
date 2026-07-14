import { readFile, writeFile, mkdir } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname } from "node:path";

export type Settings = {
  concurrencyCap?: number;
  modelAllowList?: string[];
  maxThinkingLevel?: string;
};

export const DEFAULT_CONCURRENCY_CAP = 10;
export const DEFAULT_MODEL_ALLOW_LIST = ["claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-6"];

function settingsPath(): string {
  return `${homedir()}/.ledger/settings.json`;
}

export async function loadSettings(): Promise<Settings> {
  try {
    return JSON.parse(await readFile(settingsPath(), "utf8")) as Settings;
  } catch {
    return {};
  }
}

export async function saveSettings(settings: Settings): Promise<void> {
  const path = settingsPath();
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, JSON.stringify(settings, null, 2));
}
