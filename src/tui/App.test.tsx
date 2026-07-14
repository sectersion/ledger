import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import { render } from "ink-testing-library";
import { App } from "./App.tsx";
import { demoState, DEMO_GATE_ARTIFACT } from "./demo.ts";
import { runGate } from "../gates.ts";

const tick = () => new Promise((resolve) => setImmediate(resolve));

test("kill confirm flow", async () => {
  const state = demoState();
  const { lastFrame, stdin } = render(
    <App ledgerDir="/tmp/ledger-test" initialState={state} />,
  );
  await tick();
  stdin.write("k");
  await tick();
  assert.match(lastFrame() ?? "", /kill codebase-locator\? y\/n/);
  stdin.write("n");
  await tick();
  assert.doesNotMatch(lastFrame() ?? "", /kill codebase-locator\? y\/n/);
});

test("gate approve resolves the pending gate", async () => {
  const state = demoState();
  const decision = runGate("test-gate");
  const { stdin } = render(
    <App
      ledgerDir="/tmp/ledger-test"
      initialState={state}
      pendingGateName="test-gate"
      pendingGateArtifact={DEMO_GATE_ARTIFACT}
    />,
  );
  await tick();
  stdin.write("a");
  const result = await decision;
  assert.equal(result.decision, "approve");
});

test("search filters the tree", async () => {
  const state = demoState();
  const { lastFrame, stdin } = render(
    <App ledgerDir="/tmp/ledger-test" initialState={state} />,
  );
  await tick();
  stdin.write("/");
  await tick();
  stdin.write("backend");
  await tick();
  const frame = lastFrame() ?? "";
  assert.match(frame, /backend-3/);
  assert.doesNotMatch(frame, /frontend-2/);
});
