import { appendFile, readFile, mkdir } from "node:fs/promises";
import { dirname } from "node:path";

export type JournalEntry =
  | { type: "phase"; phase: string; status: "started" | "completed" | "failed" }
  | { type: "diff"; summary: string }
  | { type: "verdict"; verdict: "pass" | "fail"; detail?: string }
  | { type: "ownership"; action: "granted" | "released"; path: string }
  | { type: "error"; message: string };

export type JournalRecord = JournalEntry & { timestamp: string };

export function journalPath(ledgerDir: string, agentId: string): string {
  return `${ledgerDir}/journals/${agentId}.jsonl`;
}

export async function appendJournal(
  ledgerDir: string,
  agentId: string,
  entry: JournalEntry,
): Promise<void> {
  const path = journalPath(ledgerDir, agentId);
  await mkdir(dirname(path), { recursive: true });
  const record: JournalRecord = { ...entry, timestamp: new Date().toISOString() };
  await appendFile(path, JSON.stringify(record) + "\n");
}

export async function readJournal(
  ledgerDir: string,
  agentId: string,
): Promise<JournalRecord[]> {
  let raw: string;
  try {
    raw = await readFile(journalPath(ledgerDir, agentId), "utf8");
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return [];
    throw err;
  }
  return raw
    .split("\n")
    .filter((line) => line.trim())
    .map((line) => JSON.parse(line) as JournalRecord);
}
