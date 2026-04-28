import { describe, it, expect, beforeAll } from "vitest";
import { chmodSync, existsSync, mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { callWire, resolveBinary } from "../src/utils/wire.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const MOCK_BIN = resolve(__dirname, "fixtures/mock-chainbench-net.mjs");

type Step =
  | { kind: "stdout"; line: string; delayMs?: number }
  | { kind: "stderr"; line: string; delayMs?: number }
  | { kind: "exit"; code?: number; delayMs?: number };

function script(steps: Step[]): string {
  return Buffer.from(JSON.stringify(steps), "utf-8").toString("base64");
}

beforeAll(() => {
  // Ensure fixture is executable on macOS / Linux even if a fresh checkout
  // landed without the executable bit.
  if (existsSync(MOCK_BIN)) {
    chmodSync(MOCK_BIN, 0o755);
  }
});

describe("resolveBinary", () => {
  it("resolveBinary_PrefersEnv", () => {
    const dir = mkdtempSync(join(tmpdir(), "cb-wire-env-"));
    const binPath = join(dir, "chainbench-net-fake");
    writeFileSync(binPath, "#!/bin/sh\nexit 0\n");
    chmodSync(binPath, 0o755);
    const savedBin = process.env.CHAINBENCH_NET_BIN;
    const savedDir = process.env.CHAINBENCH_DIR;
    try {
      process.env.CHAINBENCH_NET_BIN = binPath;
      delete process.env.CHAINBENCH_DIR;
      expect(resolveBinary()).toBe(binPath);
    } finally {
      if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
      else process.env.CHAINBENCH_NET_BIN = savedBin;
      if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
      else process.env.CHAINBENCH_DIR = savedDir;
    }
  });

  it("resolveBinary_FallsBackToChainbenchDir", () => {
    const dir = mkdtempSync(join(tmpdir(), "cb-wire-dir-"));
    const binDir = join(dir, "bin");
    mkdirSync(binDir);
    const binPath = join(binDir, "chainbench-net");
    writeFileSync(binPath, "#!/bin/sh\nexit 0\n");
    chmodSync(binPath, 0o755);
    const savedBin = process.env.CHAINBENCH_NET_BIN;
    const savedDir = process.env.CHAINBENCH_DIR;
    const savedPath = process.env.PATH;
    try {
      delete process.env.CHAINBENCH_NET_BIN;
      process.env.CHAINBENCH_DIR = dir;
      // Empty PATH so PATH lookup cannot find anything.
      process.env.PATH = "";
      expect(resolveBinary()).toBe(binPath);
    } finally {
      if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
      else process.env.CHAINBENCH_NET_BIN = savedBin;
      if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
      else process.env.CHAINBENCH_DIR = savedDir;
      if (savedPath === undefined) delete process.env.PATH;
      else process.env.PATH = savedPath;
    }
  });

  it("resolveBinary_NotFound_Throws", () => {
    const dir = mkdtempSync(join(tmpdir(), "cb-wire-none-"));
    const savedBin = process.env.CHAINBENCH_NET_BIN;
    const savedDir = process.env.CHAINBENCH_DIR;
    const savedPath = process.env.PATH;
    try {
      delete process.env.CHAINBENCH_NET_BIN;
      process.env.CHAINBENCH_DIR = dir;
      process.env.PATH = "";
      expect(() => resolveBinary()).toThrow(/chainbench-net binary not found/);
    } finally {
      if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
      else process.env.CHAINBENCH_NET_BIN = savedBin;
      if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
      else process.env.CHAINBENCH_DIR = savedDir;
      if (savedPath === undefined) delete process.env.PATH;
      else process.env.PATH = savedPath;
    }
  });
});

describe("callWire", () => {
  it("callWire_Happy_ReturnsResultEventsProgress", async () => {
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "progress", step: "load", done: 1, total: 2 }) },
      { kind: "stdout", line: JSON.stringify({ type: "progress", step: "fetch", done: 2, total: 2 }) },
      { kind: "stdout", line: JSON.stringify({ type: "event", name: "node.connected", data: { url: "http://x" } }) },
      { kind: "stdout", line: JSON.stringify({ type: "result", ok: true, data: { balance: "0x10" } }) },
    ];
    const out = await callWire(
      "node.account_state",
      { network: "local", address: "0x" + "a".repeat(40) },
      { binaryPath: MOCK_BIN, envOverrides: { MOCK_SCRIPT: script(steps) } },
    );
    expect(out.result.type).toBe("result");
    expect(out.result.ok).toBe(true);
    if (out.result.ok) {
      expect(out.result.data).toEqual({ balance: "0x10" });
    }
    expect(out.events).toHaveLength(1);
    expect(out.events[0]?.name).toBe("node.connected");
    expect(out.progress).toHaveLength(2);
    expect(out.progress[0]?.step).toBe("load");
    expect(out.progress[1]?.step).toBe("fetch");
    expect(out.exitCode).toBe(0);
  });

  it("callWire_OnError_StillReturnsResultLine", async () => {
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "progress", step: "load" }) },
      { kind: "stdout", line: JSON.stringify({
        type: "result",
        ok: false,
        error: { code: "INVALID_ARGS", message: "bad address" },
      }) },
    ];
    const out = await callWire(
      "node.account_state",
      {},
      { binaryPath: MOCK_BIN, envOverrides: { MOCK_SCRIPT: script(steps) } },
    );
    expect(out.result.ok).toBe(false);
    if (out.result.ok === false) {
      expect(out.result.error.code).toBe("INVALID_ARGS");
      expect(out.result.error.message).toBe("bad address");
    }
    expect(out.exitCode).toBe(0);
  });

  it("callWire_NoTerminator_Throws", async () => {
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "progress", step: "load" }) },
      // No result terminator before exit.
    ];
    await expect(
      callWire(
        "node.account_state",
        {},
        { binaryPath: MOCK_BIN, envOverrides: { MOCK_SCRIPT: script(steps) } },
      ),
    ).rejects.toThrow(/no result terminator/);
  });

  it("callWire_Timeout_KillsAndThrows", async () => {
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "progress", step: "stalled" }), delayMs: 60000 },
      { kind: "stdout", line: JSON.stringify({ type: "result", ok: true, data: {} }) },
    ];
    await expect(
      callWire(
        "node.account_state",
        {},
        {
          binaryPath: MOCK_BIN,
          envOverrides: { MOCK_SCRIPT: script(steps) },
          timeoutMs: 200,
        },
      ),
    ).rejects.toThrow(/timeout/);
  });

  it("callWire_PassesEnvOverrides", async () => {
    // The mock receives FOO via envOverrides; the test scenario echoes a
    // result line with a fixed marker so we know the spawn happened with the
    // provided env. (We do not need the mock to read FOO at runtime — the
    // spawn-level env propagation is what we're verifying.)
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "result", ok: true, data: { foo: "bar" } }) },
    ];
    const out = await callWire(
      "node.account_state",
      {},
      {
        binaryPath: MOCK_BIN,
        envOverrides: { MOCK_SCRIPT: script(steps), FOO: "bar" },
      },
    );
    expect(out.result.ok).toBe(true);
    if (out.result.ok) {
      expect(out.result.data).toEqual({ foo: "bar" });
    }
    // stderr from the mock contains the slog debug line and the envelope.
    expect(out.stderr).toContain("envelope");
  });

  it("callWire_NonZeroExit_WithResult_Succeeds", async () => {
    const steps: Step[] = [
      { kind: "stdout", line: JSON.stringify({ type: "result", ok: true, data: { ok: 1 } }) },
      { kind: "exit", code: 1 },
    ];
    const out = await callWire(
      "node.account_state",
      {},
      { binaryPath: MOCK_BIN, envOverrides: { MOCK_SCRIPT: script(steps) } },
    );
    expect(out.result.ok).toBe(true);
    expect(out.exitCode).toBe(1);
  });
});
