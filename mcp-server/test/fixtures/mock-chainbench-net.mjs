#!/usr/bin/env node
// mock-chainbench-net.mjs — programmable NDJSON mock for wire.test.ts.
//
// Reads a JSON envelope on stdin (single line), then emits a caller-defined
// sequence of stdout / stderr / exit steps. The scenario is supplied via the
// MOCK_SCRIPT env var as base64-encoded JSON of an array:
//   [{kind: "stdout"|"stderr"|"exit", line?: string, code?: number, delayMs?: number}]
// Default (W10= = "[]") is an empty script — process exits 0 with no output.
//
// Set MOCK_EXIT=<n> to force a non-zero exit when the script does not contain
// an explicit "exit" step.

import { stdin, stdout, stderr } from "node:process";

const script = JSON.parse(
  Buffer.from(process.env.MOCK_SCRIPT ?? "W10=", "base64").toString("utf-8"),
);

const envelopeChunks = [];
stdin.on("data", (b) => envelopeChunks.push(b));
stdin.on("end", async () => {
  const envelope = Buffer.concat(envelopeChunks).toString("utf-8").trim();
  // Echo envelope to stderr as a debug line (slog-shape JSON).
  stderr.write(
    JSON.stringify({ level: "DEBUG", msg: "envelope", envelope }) + "\n",
  );

  for (const step of script) {
    if (step.delayMs) await new Promise((r) => setTimeout(r, step.delayMs));
    if (step.kind === "stdout") stdout.write(step.line + "\n");
    else if (step.kind === "stderr") stderr.write(step.line + "\n");
    else if (step.kind === "exit") process.exit(step.code ?? 0);
  }
  process.exit(process.env.MOCK_EXIT ? Number(process.env.MOCK_EXIT) : 0);
});
