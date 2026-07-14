import React from "react";
import { render } from "ink";
import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { resumeState } from "./resume.ts";
import { App } from "./tui/App.tsx";
import { demoState, DEMO_GATE_ARTIFACT } from "./tui/demo.ts";
import { runGate } from "./gates.ts";

const ledgerDirArg = process.argv[2] ?? process.env.LEDGER_DIR;

async function main() {
  if (ledgerDirArg) {
    const state = await resumeState(ledgerDirArg);
    render(<App ledgerDir={ledgerDirArg} initialState={state} />);
    return;
  }

  const ledgerDir = await mkdtemp(join(tmpdir(), "ledger-demo-"));
  const state = demoState();
  runGate("research").catch(() => {});
  render(
    <App
      ledgerDir={ledgerDir}
      initialState={state}
      pendingGateName="research"
      pendingGateArtifact={DEMO_GATE_ARTIFACT}
    />,
  );
}

main();
