import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  StopArgs,
  _stopHandler,
  StatusArgs,
  _statusHandler,
  InitArgs,
  _initHandler,
  _startHandler,
  _restartHandler,
  CleanArgs,
  _cleanHandler,
} from "../src/tools/lifecycle.js";

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
  // Ensure the mock fixture is executable on a fresh checkout.
  if (existsSync(MOCK_BIN)) {
    chmodSync(MOCK_BIN, 0o755);
  }
});

// _stopHandler calls callWire() which calls resolveBinary() — there is no
// binaryPath option exposed at the handler boundary. Tests inject the mock
// by setting CHAINBENCH_NET_BIN, MOCK_SCRIPT (mock fixture scenario), and
// CHAINBENCH_DIR (cleared so resolveBinary can't pick a real binary). Saved
// values are restored in afterEach to keep tests isolated.
describe("chainbench_stop handler", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  it("_Happy", async () => {
    const data = { network: "local", stdout: "stopped 4 nodes" };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _stopHandler({});
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain("stopped 4 nodes");
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'force', 'timeout') trip a
    // structured INVALID_ARGS at the zod boundary instead of being silently
    // stripped — fail-loud beats silent strip on an LLM-driven surface.
    expect(() =>
      StopArgs.parse({ network: "local", extra: "bar" }),
    ).toThrow();
  });

  it("_WireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "UPSTREAM_ERROR", message: "chainbench stop exited 1" },
        }),
      },
    ]);
    const out = await _stopHandler({ network: "local" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("chainbench stop exited 1");
  });
});

// chainbench_status mirrors the stop handler's wire-routing pattern. The
// wire envelope's `data` is the parsed JSON from `chainbench status --json`
// (see network.status Go handler in Sprint 5c.4.1 Task 3); formatWireResult
// pretty-prints the map, so assertions look for known keys in the rendered
// text. Same env snapshot/restore convention as the stop block above.
describe("chainbench_status handler", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  it("_StatusHappy", async () => {
    const data = {
      nodes: [{ id: "node1", running: true, block_height: 42 }],
      healthy: true,
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _statusHandler({});
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain("healthy");
    expect(out.content[0]?.text).toContain("block_height");
  });

  it("_StatusStrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'verbose', 'json') trip a
    // structured zod error instead of being silently stripped — same
    // rationale as StopArgs above.
    expect(() =>
      StatusArgs.parse({ network: "local", extra: "bar" }),
    ).toThrow();
  });

  it("_StatusWireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "UPSTREAM_ERROR", message: "chainbench status exited 1" },
        }),
      },
    ]);
    const out = await _statusHandler({ network: "local" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("chainbench status exited 1");
  });
});

// chainbench_init/start/restart route through the Go wire (network.init /
// network.start_all / network.restart). Same mock-fixture injection as the
// stop/status blocks: the mock chainbench-net replays a wire result line, and
// boundary-rejection cases return before any spawn (no MOCK_SCRIPT needed).
describe("chainbench_init/start/restart handlers", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  function okResult(stdout: string): void {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data: { stdout } }),
      },
    ]);
  }

  it("_InitDefaultsProfileAndRoutesToWire", async () => {
    okResult("initialized");
    const out = await _initHandler({ profile: "default" });
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("initialized");
  });

  it("_InitRejectsBadProfileBeforeSpawn", async () => {
    const out = await _initHandler({ profile: "../etc/passwd" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("invalid profile name");
  });

  it("_InitRejectsRelativeBinaryPath", async () => {
    const out = await _initHandler({ profile: "default", binary_path: "rel/gstable" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("binary_path must be an absolute path");
  });

  it("_InitStrictRejectsUnknownKeys", () => {
    expect(() => InitArgs.parse({ profile: "default", bogus: 1 })).toThrow();
  });

  it("_StartRoutesToWire", async () => {
    okResult("started");
    const out = await _startHandler({ binary_path: "/opt/gstable" });
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("started");
  });

  it("_StartRejectsRelativeProjectRoot", async () => {
    const out = await _startHandler({ project_root: "rel/root" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("project_root must be an absolute path");
  });

  it("_RestartRoutesToWire", async () => {
    okResult("restarted");
    const out = await _restartHandler({});
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("restarted");
  });
});

// chainbench_clean routes through the network.clean wire handler. It takes no
// arguments (the Go handler removes node data regardless of profile/root), so
// the only boundary contract is .strict() rejecting hallucinated keys. Same
// mock-fixture injection convention as the blocks above.
describe("chainbench_clean handler", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  it("_CleanRoutesToWire", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data: { stdout: "cleaned node data" } }),
      },
    ]);
    const out = await _cleanHandler({});
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("cleaned node data");
  });

  it("_CleanStrictRejectsUnknownKeys", () => {
    expect(() => CleanArgs.parse({ force: true })).toThrow();
  });

  it("_CleanWireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "UPSTREAM_ERROR", message: "chainbench clean exited 1" },
        }),
      },
    ]);
    const out = await _cleanHandler({});
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("chainbench clean exited 1");
  });
});
