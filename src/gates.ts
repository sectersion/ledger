export type GateDecision = { decision: "approve" | "reject"; comment?: string };

const pending = new Map<string, (d: GateDecision) => void>();

/** Blocks until resolveGate(name, ...) is called externally (e.g. by the TUI). */
export function runGate(name: string): Promise<GateDecision> {
  return new Promise((resolve) => {
    pending.set(name, resolve);
  });
}

export function resolveGate(name: string, decision: GateDecision): boolean {
  const resolve = pending.get(name);
  if (!resolve) return false;
  pending.delete(name);
  resolve(decision);
  return true;
}

export function hasPendingGate(name: string): boolean {
  return pending.has(name);
}
