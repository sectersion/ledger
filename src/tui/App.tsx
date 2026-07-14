import React, { useEffect, useMemo, useState } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import type { AgentState, ResumeState } from "../resume.ts";
import { killWorker } from "../worker.ts";
import { appendJournal, type JournalRecord } from "../journal.ts";
import { resolveGate, runGate, type GateDecision } from "../gates.ts";

const PHASE_ORDER = ["research", "plan", "implement", "validate", "review", "ship"];

type Row = { kind: "phase"; phase: string } | { kind: "agent"; agent: AgentState };

type Modal =
  | { kind: "killConfirm" }
  | { kind: "btw"; text: string }
  | { kind: "search"; text: string }
  | { kind: "gate"; name: string; artifact: string; scroll: number; comment: string; editing: boolean };

type Toast = { id: number; message: string };

function entryLine(e: JournalRecord): string {
  const t = e.timestamp.slice(11, 19);
  switch (e.type) {
    case "phase":
      return `[${t}] phase ${e.phase}: ${e.status}`;
    case "diff":
      return `[${t}] ${e.summary}`;
    case "verdict":
      return `[${t}] verdict: ${e.verdict}${e.detail ? ` (${e.detail})` : ""}`;
    case "ownership":
      return `[${t}] ownership ${e.action}: ${e.path}`;
    case "error":
      return `[${t}] error: ${e.message}`;
  }
}

function statusMark(agent: AgentState, paused: boolean): string {
  if (paused) return "⏸";
  if (agent.status === "started") return "●";
  if (agent.status === "completed") return "✓";
  if (agent.status === "failed") return "✗";
  return "⏳";
}

export function App({
  ledgerDir,
  initialState,
  pendingGateName,
  pendingGateArtifact,
}: {
  ledgerDir: string;
  initialState: ResumeState;
  pendingGateName?: string;
  pendingGateArtifact?: string;
}) {
  const [agents, setAgents] = useState<AgentState[]>(initialState.agents);
  const [paused, setPaused] = useState<Set<string>>(new Set());
  const [cursor, setCursor] = useState(0);
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(agents[0]?.agentId ?? null);
  const [modal, setModal] = useState<Modal | null>(
    pendingGateName
      ? { kind: "gate", name: pendingGateName, artifact: pendingGateArtifact ?? "", scroll: 0, comment: "", editing: false }
      : null,
  );
  const [toast, setToast] = useState<Toast | null>(null);

  const { stdout } = useStdout();
  const [size, setSize] = useState({ columns: stdout.columns || 80, rows: stdout.rows || 24 });
  useEffect(() => {
    stdout.write("\x1b[?1049h\x1b[2J\x1b[H");
    const onResize = () => setSize({ columns: stdout.columns || 80, rows: stdout.rows || 24 });
    stdout.on("resize", onResize);
    return () => {
      stdout.off("resize", onResize);
      stdout.write("\x1b[?1049l");
    };
  }, [stdout]);

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast((t) => (t?.id === toast.id ? null : t)), 2500);
    return () => clearTimeout(id);
  }, [toast]);

  function notify(message: string) {
    setToast({ id: Date.now(), message });
  }

  const phases = useMemo(() => {
    const present = new Set(agents.map((a) => a.phase ?? "unphased"));
    return PHASE_ORDER.filter((p) => present.has(p)).concat(present.has("unphased") ? ["unphased"] : []);
  }, [agents]);

  const searchText = modal?.kind === "search" ? modal.text : "";
  const rows: Row[] = useMemo(() => {
    const out: Row[] = [];
    for (const phase of phases) {
      const inPhase = agents.filter((a) => (a.phase ?? "unphased") === phase);
      const matching = searchText
        ? inPhase.filter((a) => a.agentId.toLowerCase().includes(searchText.toLowerCase()))
        : inPhase;
      if (searchText && matching.length === 0) continue;
      out.push({ kind: "phase", phase });
      if (searchText || !collapsed.has(phase)) {
        for (const a of matching) out.push({ kind: "agent", agent: a });
      }
    }
    return out;
  }, [phases, agents, collapsed, searchText]);

  useEffect(() => {
    if (cursor >= rows.length) setCursor(Math.max(0, rows.length - 1));
  }, [rows, cursor]);

  const selectedAgent = agents.find((a) => a.agentId === selectedAgentId) ?? null;

  useInput((input, key) => {
    if (modal?.kind === "gate") {
      handleGateInput(input, key);
      return;
    }
    if (modal?.kind === "killConfirm") {
      if (input === "y") {
        if (selectedAgent) {
          killWorker(selectedAgent.agentId);
          notify(`killed ${selectedAgent.agentId}`);
        }
        setModal(null);
      } else if (input === "n" || key.escape) {
        setModal(null);
      }
      return;
    }
    if (modal?.kind === "btw") {
      if (key.return) {
        submitBtw(modal.text);
      } else if (key.escape) {
        setModal(null);
      } else if (key.backspace || key.delete) {
        setModal({ ...modal, text: modal.text.slice(0, -1) });
      } else if (input) {
        setModal({ ...modal, text: modal.text + input });
      }
      return;
    }
    if (modal?.kind === "search") {
      if (key.return || key.escape) {
        setModal(null);
      } else if (key.backspace || key.delete) {
        setModal({ ...modal, text: modal.text.slice(0, -1) });
      } else if (input) {
        setModal({ ...modal, text: modal.text + input });
      }
      return;
    }

    if (key.upArrow) setCursor((c) => Math.max(0, c - 1));
    else if (key.downArrow) setCursor((c) => Math.min(rows.length - 1, c + 1));
    else if (key.leftArrow || key.rightArrow) {
      const row = rows[cursor];
      if (row?.kind === "phase") toggleCollapse(row.phase);
    } else if (input === " ") {
      const row = rows[cursor];
      if (row?.kind === "agent") setSelectedAgentId(row.agent.agentId);
    } else if (input === "k") {
      if (selectedAgent) setModal({ kind: "killConfirm" });
    } else if (input === "p") {
      if (selectedAgent) {
        setPaused((prev) => {
          const next = new Set(prev);
          if (next.has(selectedAgent.agentId)) next.delete(selectedAgent.agentId);
          else next.add(selectedAgent.agentId);
          return next;
        });
        notify(`paused ${selectedAgent.agentId}`);
      }
    } else if (input === "b") {
      if (selectedAgent) setModal({ kind: "btw", text: "" });
    } else if (input === "/") {
      setModal({ kind: "search", text: "" });
    }
  });

  function toggleCollapse(phase: string) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(phase)) next.delete(phase);
      else next.add(phase);
      return next;
    });
  }

  function submitBtw(message: string) {
    if (!selectedAgent || !message.trim()) {
      setModal(null);
      return;
    }
    killWorker(selectedAgent.agentId);
    appendJournal(ledgerDir, selectedAgent.agentId, { type: "error", message: `/btw: ${message}` }).catch(() => {});
    notify(`relayed to ${selectedAgent.agentId}`);
    setModal(null);
  }

  function handleGateInput(input: string, key: { escape?: boolean; return?: boolean; backspace?: boolean; delete?: boolean; upArrow?: boolean; downArrow?: boolean }) {
    if (modal?.kind !== "gate") return;
    if (modal.editing) {
      if (key.return) {
        setModal({ ...modal, editing: false });
      } else if (key.backspace || key.delete) {
        setModal({ ...modal, comment: modal.comment.slice(0, -1) });
      } else if (input) {
        setModal({ ...modal, comment: modal.comment + input });
      }
      return;
    }
    if (input === "j" || key.downArrow) {
      setModal({ ...modal, scroll: modal.scroll + 1 });
    } else if (input === "k" || key.upArrow) {
      setModal({ ...modal, scroll: Math.max(0, modal.scroll - 1) });
    } else if (input === "e") {
      setModal({ ...modal, editing: true });
    } else if (input === "a" || input === "r") {
      const decision: GateDecision = {
        decision: input === "a" ? "approve" : "reject",
        comment: modal.comment || undefined,
      };
      resolveGate(modal.name, decision);
      notify(`gate ${modal.name}: ${decision.decision}`);
      setModal(null);
    }
  }

  const phase = selectedAgent?.phase ?? "-";
  const substage = selectedAgent
    ? `${selectedAgent.agentId} ${selectedAgent.status ?? "queued"}`
    : "none";
  const running = agents.filter((a) => a.status === "started" && !paused.has(a.agentId)).length;
  const gateState = modal?.kind === "gate" ? `pending (${modal.name})` : "none pending";

  return (
    <Box flexDirection="column" width={size.columns} height={size.rows}>
      <Text>
        ledger · stage: {phase} · substage: {substage}
      </Text>
      <Box borderStyle="single" flexDirection="row" flexGrow={1}>
        <Box flexDirection="column" width="40%" borderStyle="single">
          <Text bold> AGENTS</Text>
          {rows.map((row, i) => {
            const isCursor = i === cursor;
            if (row.kind === "phase") {
              const mark = searchText ? "▾" : collapsed.has(row.phase) ? "▸" : "▾";
              return (
                <Text key={`p-${row.phase}`} inverse={isCursor}>
                  {mark} {row.phase}
                </Text>
              );
            }
            const a = row.agent;
            const sel = a.agentId === selectedAgentId ? "*" : " ";
            return (
              <Text key={a.agentId} inverse={isCursor}>
                {sel} {statusMark(a, paused.has(a.agentId))} {a.agentId}
              </Text>
            );
          })}
        </Box>
        <Box flexDirection="column" width="60%" borderStyle="single">
          <Text bold> agent: {selectedAgent?.agentId ?? "-"}</Text>
          {selectedAgent ? (
            <>
              <Text>status: {selectedAgent.status ?? "queued"}{paused.has(selectedAgent.agentId) ? " (paused)" : ""}</Text>
              {selectedAgent.entries.map((e, i) => (
                <Text key={i}>{entryLine(e)}</Text>
              ))}
            </>
          ) : (
            <Text>no agent selected</Text>
          )}
        </Box>
      </Box>
      <Text>
        workers: {running}/10 · gate: {gateState} · [k]ill [p]ause [b]tw [/]search
      </Text>
      {toast ? <Text color="yellow">{toast.message}</Text> : null}
      {modal?.kind === "killConfirm" ? (
        <Text color="red">kill {selectedAgent?.agentId}? y/n</Text>
      ) : null}
      {modal?.kind === "btw" ? <Text>/btw: {modal.text}_</Text> : null}
      {modal?.kind === "search" ? <Text>/search: {modal.text}_</Text> : null}
      {modal?.kind === "gate" ? (
        <Box flexDirection="column" borderStyle="double">
          <Text bold>gate: {modal.name}</Text>
          {modal.artifact
            .split("\n")
            .slice(modal.scroll, modal.scroll + 8)
            .map((line, i) => <Text key={i}>{line}</Text>)}
          <Text>comment: {modal.comment}{modal.editing ? "_" : ""}</Text>
          <Text>[a]pprove [r]eject [e]dit-comment [j/k] scroll</Text>
        </Box>
      ) : null}
    </Box>
  );
}
