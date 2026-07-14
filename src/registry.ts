import { readFile, writeFile, mkdir } from "node:fs/promises";
import { dirname } from "node:path";

export type Registry = Record<string, string>; // path -> owning agentId

function registryPath(ledgerDir: string): string {
  return `${ledgerDir}/registry.json`;
}

export async function loadRegistry(ledgerDir: string): Promise<Registry> {
  try {
    return JSON.parse(await readFile(registryPath(ledgerDir), "utf8")) as Registry;
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return {};
    throw err;
  }
}

export async function saveRegistry(ledgerDir: string, registry: Registry): Promise<void> {
  const path = registryPath(ledgerDir);
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, JSON.stringify(registry, null, 2));
}

/** Grants ownership if unowned/self-owned; returns whether the grant succeeded. */
export async function requestOwnership(
  ledgerDir: string,
  path: string,
  agentId: string,
): Promise<boolean> {
  const registry = await loadRegistry(ledgerDir);
  const owner = registry[path];
  if (owner && owner !== agentId) return false;
  registry[path] = agentId;
  await saveRegistry(ledgerDir, registry);
  return true;
}

export async function releaseOwnership(
  ledgerDir: string,
  path: string,
  agentId: string,
): Promise<void> {
  const registry = await loadRegistry(ledgerDir);
  if (registry[path] === agentId) {
    delete registry[path];
    await saveRegistry(ledgerDir, registry);
  }
}
