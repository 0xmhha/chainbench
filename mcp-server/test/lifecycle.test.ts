import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { StopArgs, _stopHandler } from "../src/tools/lifecycle.js";

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
